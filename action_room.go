package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"strconv"
	"unicode"
)

type room struct {
	title string
	synonyms synonyms
	commands []command
}

type synonyms map[string][]string
func (s synonyms) add(words []string) {
	/*
	wordSet := makeStringSet(words)
	for _, word := range wordSet {
		s[word] = s[word].union(wordSet)
	}
	*/
	for i, word := range words {
		s[word] = append(s[word], append(words[:i:i], words[i+1:]...)...)
	}
}

type command struct{ action //
	patternIsExact bool
	patternWords []patternWord
	embeddedInteractions []action
	initialIndex int16 //used for sorting 
}

type patternWord struct {
	mask patternWordMask
	string
}

type action struct {
	conditionFlags, setFlags []flag
	randomlySetFlags []randomlySetFlag
	characterIncrement int8
	outputText string
	roomToGoto string
}

type patternWordMask byte
const (
	firstCharWasPercent patternWordMask = iota + 1
	lastCharWasPercent
)

type flag struct {
	key string
	value bool
}

type randomlySetFlag struct {
	key string
	probability uint8
}

func loadRoom() *room {
	var (
		stack = stack{topLevel}
		scanner = bufio.NewScanner(os.Stdin)
		line string
		lineFields = fieldsN(line, 2)
		room = room{synonyms: synonyms{}}
		currentCommand *command
		currentAction *action
		interactions *[]action
		addNewCommand = func(string string, patternIsExact bool) {
			patternWords := []patternWord(nil)
			for _, word := range strings.Fields(strings.ToLower(string)) {
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
			room.commands = append(room.commands,
				command{patternWords: patternWords, patternIsExact: patternIsExact,
					action: action{}})
			i := len(room.commands)-1
			room.commands[i].initialIndex = int16(i)
			currentCommand = &room.commands[i]
			currentAction = &currentCommand.action
			interactions = &currentCommand.embeddedInteractions
		}
		parseFlags = func(strings []string) (flags []flag) {
			for _, string := range strings {
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
		stringsBuilder strings.Builder
	)
	for {
		switch stack.peek() {
		case topLevel:
			switch lineFields[0] {
			case "^": //ignore and consume an empty line
			case "^РН":
				room.title = lineFields[2]
				stack.replaceTop(stack.peek()+1)
			default:
				stack.replaceTop(stack.peek()+1)
				continue
			}
		case topLevel1:
			switch lineFields[0] {
			case "^ОП":
				addNewCommand("о", true)
				(*currentAction).conditionFlags = parseFlags(strings.Fields(lineFields[2]))
				stack.push(innerLevel)
			case "^Т":
				addNewCommand(lineFields[2], false)
				stack.push(innerLevel)
			case "^": //ignore and consume an empty line
			default:
				if lineFields[1]=="=" {
					room.synonyms.add(append(
						strings.FieldsFunc(strings.ToLower(lineFields[2]), func(r rune) bool {
							if r==',' || unicode.IsSpace(r) {
								return true
							}
							return false
						}),
						strings.ToLower(strings.TrimPrefix(lineFields[0],"^")),
					))
				} else if lineFields[0]=="^ДИАЛ" {
					addNewCommand(line, false)
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
				(*currentAction).conditionFlags = parseFlags(strings.Fields(lineFields[2]))
			case "^Д":
				(*currentAction).setFlags = parseFlags(strings.Fields(lineFields[2]))
			case "^ВЕР":
				lineFields1Fields := strings.Fields(lineFields[1])
				probability, _ := strconv.Atoi(lineFields1Fields[1])
				(*currentAction).randomlySetFlags = append((*currentAction).randomlySetFlags,
					randomlySetFlag{key:lineFields1Fields[0], probability:uint8(probability)})
			case "^ИД":
				(*currentAction).roomToGoto = lineFields[2]
			case "^СС":
				(*currentAction).characterIncrement++
			case "^СТ":
				(*currentAction).characterIncrement--
			case "^ДИ", "^", "^КДИАЛ":
				(*currentAction).outputText = stringsBuilder.String()
				stringsBuilder.Reset()
				stack.removeTop()
				continue
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
	sort.Slice(room.commands, func(i, j int) bool {
		commands := room.commands
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
	fmt.Printf("%q\n", room.synonyms)
	return &room
}
