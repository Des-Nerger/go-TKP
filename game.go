package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"strconv"
	"time"
	"unicode"
)

type game struct {
	CurrentRoomId string
	СС, СТ uint8
	FlagsMap flagsMap

	room0, currentRoom room
	currentDialog *command
	rand *rand.Rand
	savefile *os.File
	saveEncoder interface {Encode(interface{}) error}
}

type room struct {
	title string
	synonyms synonyms
	commands []command
}

type flagsMap map[string]struct{}
func (fm flagsMap) satisfiesConditions(conditionFlags flags) bool {
	for _, conditionFlag := range conditionFlags {
		_, ok := fm[conditionFlag.key]
		if ok != conditionFlag.value {return false}
	}
	return true
}
func (fm flagsMap) setFlag(key string, value bool) {
	if value {
		fm[key] = struct{}{}
	} else {
		delete(fm, key)
	}
}
func (fm flagsMap) updateCharacterFlags(СС, СТ uint8) {
	isExtraordinary := func(force, oppositeForce uint8) bool {
		if oppositeForce==0 {return force>4}
		return force>4*oppositeForce
	}
	const (isGood="0"; isMediocre="3000")
	set := fm.setFlag
	if СС >= СТ {
		set(isGood, true)
		set(isMediocre, !isExtraordinary(СС, СТ))
	} else {
		set(isGood, false)
		set(isMediocre, !isExtraordinary(СТ, СС))
	}
}

type synonyms map[string][]string
func (s synonyms) add(words []string) {
	for i, word := range words {
		s[word] = append(s[word], append(words[:i:i], words[i+1:]...)...)
	}
}

type command struct{ action //
	patternIsExact bool
	patternWords []patternWord
	embeddedInteractions embeddedInteractions
	initialIndex uint16 //used for sorting
}

type patternWord struct {
	mask patternWordMask
	string
}

type embeddedInteractions []action
func (ei *embeddedInteractions) forEachSatisfiedBy(fm flagsMap,
	callback func(choiceNº int, _ *action) (breakLoop bool),
) {
	//i:=0
	for j, _ := range *ei {
		action := &(*ei)[j]
		//*
		if callback(
			func() int {
				if fm.satisfiesConditions(action.conditionFlags) {
					//i++
					return j+1 //i
				}
				return -(j+1) //-1
			} (),
			action,
		) {
			break
		}
		/*/
		if fm.satisfiesConditions(action.conditionFlags) {
			if callback(j+1, action) {break}
		}
		//*/
	}
}

type action struct {
	conditionFlags, setFlags flags
	randomlySetFlags []randomlySetFlag
	ССCount, СТCount uint8
	outputText string
	roomIdToGoto string
}

type patternWordMask byte
const (
	firstCharWasPercent patternWordMask = iota + 1
	lastCharWasPercent
)

type flags []flag
func parseFlags(string string) (flags flags) {
	for _, string := range strings.Fields(string) {
		flags = append(flags,
			func() flag {
				if string[0] == '-' {
					return flag{key: string[1:], value: false}
				}
				return flag{key: string, value: true}
			} (),
		)
	}
	return
}
func (flags flags) String() string {
	if len(flags)==0 {return ""}
	var strlen, i int
	for {
		flag := flags[i]
		strlen += func()int{if !flag.value {return 1}; return 0}() + len(flag.key)
		i++; if i>=len(flags) {break}
		strlen += 1
	}
	sb := strings.Builder{}; sb.Grow(strlen)
	i = 0
	for {
		flag := flags[i]
		if !flag.value {sb.WriteByte('-')}
		sb.WriteString(flag.key)
		i++; if i>=len(flags) {break}
		sb.WriteByte(' ')
	}
	return sb.String()
}

type flag struct {
	key string
	value bool
}

type randomlySetFlag struct {
	key string
	probability uint8
}

func (g *game) init() *game {
	g.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	g.room0.synonyms = synonyms{}
	g.room0.load("0")
	fmt.Printf("%s.\n\n", g.room0.title)

	var err error
	g.savefile, err = os.OpenFile("savefile", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	var bootstrapCommand command
	if stat,_:=g.savefile.Stat(); stat.Size()==0 {
		g.FlagsMap = flagsMap{}
		g.CurrentRoomId = "1"
		//hacky but faster than *g.findCommand([]string{""}) :
		bootstrapCommand = g.room0.commands[len(g.room0.commands)-1]
		bootstrapCommand.outputText = ""
	} else {
		d:=json.NewDecoder(g.savefile); d.Decode(g)
	}
	e:=json.NewEncoder(g.savefile); e.SetIndent("", "\t")
	g.saveEncoder = e
	g.FlagsMap.updateCharacterFlags(g.СС, g.СТ)
	bootstrapCommand.roomIdToGoto = g.CurrentRoomId
	g.executeCommand(&bootstrapCommand)
	return g
}

func (g *game) save() {
	g.savefile.Truncate(0)
	g.savefile.Seek(0, 0)
	g.saveEncoder.Encode(g)
	g.savefile.Sync()
}

func (g *game) saveAndClose() {
	g.save()
	g.savefile.Close()
}

func (g *game) executeAction(action *action) {
	for _, flag := range action.setFlags {
		g.FlagsMap.setFlag(flag.key, flag.value)
	}
	for _, flag := range action.randomlySetFlags {
		g.FlagsMap.setFlag(flag.key, g.rand.Intn(100) < int(flag.probability))
	}
	if action.ССCount > 0 || action.СТCount > 0 {
		g.СС+=action.ССCount; g.СТ+=action.СТCount
		g.FlagsMap.updateCharacterFlags(g.СС, g.СТ)
	}
	if action.outputText != "" {
		fmt.Println(action.outputText)
	}
	if action.roomIdToGoto != "" {
		g.CurrentRoomId = action.roomIdToGoto
		g.save()
		g.currentRoom = room{synonyms: make(synonyms, len(g.room0.synonyms))}
		for key, synonyms := range g.room0.synonyms {
			g.currentRoom.synonyms[key] = synonyms
		}
		g.currentRoom.load(g.CurrentRoomId)
		if g.currentRoom.title != "" {
			fmt.Printf("Место действия: %v.\n", g.currentRoom.title)
		}
		g.executeCommand(g.findCommand([]string{"описание"}))
	}
}

func (g *game) executeCommand(command *command) {
	g.executeAction(&command.action)
	if len(command.embeddedInteractions)>0 && command.roomIdToGoto=="" {
		//fmt.Fprintf(os.Stderr, "%#v\n\n", command)
		print := func(i int, string string) {
			fmt.Printf("Номер %v: %v\n", i, string)
		}
		isThereAnySatisfied := false
		command.embeddedInteractions.forEachSatisfiedBy(g.FlagsMap,
			func(choiceNº int, action *action) (breakLoop bool) {
				if !(choiceNº < 0) {
					isThereAnySatisfied = true
				} else {
					return
				}
				print(choiceNº, action.outputText[:strings.IndexByte(action.outputText, '\n')])
				return
			},
		)
		if !isThereAnySatisfied && !command.patternIsExact {
			g.executeCommand(g.findCommand([]string{""}))
		} else {
			//print(0, "<Закончить диалог>")
			fmt.Println()
			g.currentDialog = command
		}
	}
}

func (g *game) findCommand(inputWords []string) *command {
	if g.currentDialog != nil {
		switch len(inputWords) {
		case 0:
			return &command{embeddedInteractions: g.currentDialog.embeddedInteractions}
		case 1:
			wantedChoiceNº, err := strconv.ParseUint(inputWords[0], 10, strconv.IntSize-1)
			if err == nil {
				var wantedAction *action
				g.currentDialog.embeddedInteractions.forEachSatisfiedBy(g.FlagsMap,
					func(choiceNº int, action *action) (breakLoop bool) {
						if choiceNº == int(wantedChoiceNº) {
							wantedAction = action
							breakLoop = true
						}
						return
					},
				)
				//command := &command{embeddedInteractions: g.currentDialog.embeddedInteractions}
				if wantedAction != nil {
					return &command{action: *wantedAction}
					//command.action = *wantedAction; return command
				}
				return &command{action: action{outputText: "Этот пункт не доступен.\n"}}
				//command.action = action{outputText: "Такой пункт не доступен.\n"}; return command
			}
		}
		g.currentDialog = nil
	}
	for _, r := range [...]room{g.currentRoom, g.room0} {
	commandLoop:
		for _, command := range r.commands {
			if bool(!g.FlagsMap.satisfiesConditions(command.conditionFlags) ||
			        command.patternIsExact && len(command.patternWords) != len(inputWords),
			) {continue}
		patternWordLoop:
			for _, patternWord := range command.patternWords {
				matches := func() func(string, string) bool {
					switch patternWord.mask {
					case firstCharWasPercent | lastCharWasPercent: return strings.Contains
					case firstCharWasPercent: return strings.HasSuffix
					case lastCharWasPercent: return strings.HasPrefix
					default: return func(a, b string) bool {return a == b}
					}
				} ()
				synonyms := append(r.synonyms[patternWord.string], patternWord.string)
				for _, inputWord := range inputWords {
					for _, synonym := range synonyms {
						if matches(inputWord, synonym) {
							continue patternWordLoop
						}
					}
				}
				continue commandLoop
			}
			return &command
		}
	}
	return nil
}

func (r *room) load(roomId string) {
	file, err := os.Open("room" + roomId + ".txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var (
		stack = stack{topLevel}
		scanner = bufio.NewScanner(file)
		line string
		lineFields = fieldsN(line, 2)
		currentCommand *command
		currentAction *action
		interactions *embeddedInteractions
		setPatternWords = func(string string) {
			patternWords := []patternWord(nil)
			for _, word := range strings.Fields( strings.ToLower( string ) ) {
				patternWord := patternWord{string: word}
				if patternWord.string[0] == '%' {
					patternWord.mask |= firstCharWasPercent
					patternWord.string = patternWord.string[1:]
				}
				lastIndex := len(patternWord.string)-1
				if patternWord.string[lastIndex] == '%' {
					patternWord.mask |= lastCharWasPercent
					patternWord.string = patternWord.string[:lastIndex]
				}
				patternWords = append(patternWords, patternWord)
			}
			currentCommand.patternWords = patternWords
		}
		addNewCommand = func(string string, patternIsExact bool) {
			r.commands = append(r.commands, command{patternIsExact: patternIsExact})
			i := len(r.commands)-1
			currentCommand = &r.commands[i]
			currentCommand.initialIndex = uint16(i)
			setPatternWords(string)
			currentAction = &currentCommand.action
			interactions = &currentCommand.embeddedInteractions
		}
		stringsBuilder strings.Builder
	)
	for {
		switch stack.peek() {
		case topLevel:
			switch lineFields[0] {
			case "^": //ignore and consume an empty line
			case "^РН":
				r.title = lineFields[2]
				stack.replaceTop(stack.peek()+1)
			default:
				stack.replaceTop(stack.peek()+1)
				continue
			}
		case topLevel1:
			switch lineFields[0] {
			case "^ОП":
				addNewCommand("описание", true)
				(*currentAction).conditionFlags = parseFlags(lineFields[2])
				stack.push(innerLevel)
			case "^Т":
				addNewCommand(lineFields[2], false)
				stack.push(innerLevel)
			case "^": //ignore and consume an empty line
			default:
				if lineFields[1]=="=" {
					r.synonyms.add(append(
						strings.FieldsFunc( strings.ToLower( lineFields[2] ) , func(r rune) bool {
							if r==',' || unicode.IsSpace(r) {
								return true
							}
							return false
						}),
						strings.ToLower(strings.TrimPrefix(lineFields[0],"^")),
					))
				} else if lineFields[0]=="^ДИАЛ" {
					addNewCommand(line, func() bool {
						if lineFields[2]=="АВТО" {return true}
						return false
					} ())
					stack.push(ДИАЛ)
				} else {
					fmt.Fprintf(os.Stderr, "topLevel1: warning: the %q line is ignored\n", lineFields[0])
				}
			}
		case ДИАЛ:
			switch lineFields[0] {
			case "^КДИАЛ":
				stack.removeTop()
			case "^": //ignore and consume an empty line
			case "^ДИ":
				*interactions = append(*interactions, action{})
				currentAction = &(*interactions)[len(*interactions)-1]
				stack.push(innerLevel)
			default:
				stack.push(innerLevel)
				continue
			}
		case innerLevel:
			switch lineFields[0] {
			case "^Л":
				(*currentAction).conditionFlags = parseFlags(lineFields[2])
			case "^Д":
				(*currentAction).setFlags = parseFlags(lineFields[2])
			case "^ВЕР":
				lineFields2Fields := strings.Fields(lineFields[2])
				probability, _ := strconv.Atoi(lineFields2Fields[1])
				(*currentAction).randomlySetFlags = append((*currentAction).randomlySetFlags,
					randomlySetFlag{key:lineFields2Fields[0], probability:uint8(probability)})
			case "^ИД":
				(*currentAction).roomIdToGoto = lineFields[2]
			case "^СС":
				(*currentAction).ССCount++
			case "^СТ":
				(*currentAction).СТCount++
			case "^ДИ", "^", "^КДИАЛ":
				(*currentAction).outputText = stringsBuilder.String()
				stringsBuilder.Reset()
				stack.removeTop()
				continue
			case "^Т":
				if currentAction == &currentCommand.action {
					setPatternWords(lineFields[2])
					break
				}
				fallthrough
			default:
				stringsBuilder.WriteString(line)
				stringsBuilder.WriteByte('\n')
			}
		}
		if scanner.Scan() {
			line = scanner.Text()
		} else {
			if lineFields[0] == "^" {break}
			line = ""
		}
		lineFields = fieldsN(line, func() int{if len(stack)==1 {return 2}; return 1}())
	}

	sign := func(x int) int {
		if x < 0 {return -1}
		if x > 0 {return +1}
		return x
	}
	sort.Slice(r.commands, func(i, j int) bool {
		commands := r.commands
		switch sign(len(commands[j].patternWords) - len(commands[i].patternWords)) {
		case -1: return true
		case +1: return false
		case 0:
			switch sign(len(commands[j].conditionFlags) - len(commands[i].conditionFlags)) {
			case -1: return true
			case +1: return false
			case 0: return commands[j].initialIndex < commands[i].initialIndex
			}
			fallthrough
		default:
			panic("!(-1 <= sign <= +1)")
		}
	})
}
