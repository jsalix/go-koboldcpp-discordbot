package main

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/jsalix/go-koboldcpp-discordbot/api"
	"github.com/jsalix/go-koboldcpp-discordbot/prompt"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	DISCORD_TOKEN string
	API_URL       string
	PERSONA       string
)

var PERSONA_DESC string
var SYSTEM_PROMPT string

var Api *api.KoboldClient

func main() {
	// load vars from .env
	err := godotenv.Load()
	if err != nil {
		fmt.Println("unable to load .env")
		return
	}

	DISCORD_TOKEN = os.Getenv("DISCORD_TOKEN")
	API_URL = os.Getenv("API_URL")
	PERSONA = os.Getenv("PERSONA")

	// load persona file
	personaFile, err := os.ReadFile("prompt/persona/" + PERSONA + ".txt")
	if err != nil {
		fmt.Println("unable to load persona file")
		return
	}

	PERSONA_DESC = string(personaFile)

	if PERSONA_DESC == "" {
		fmt.Println("WARNING! persona file appears to be empty")
	}

	// load system prompt
	systemPromptFile, err := os.ReadFile("prompt/system/default.txt")
	if err != nil {
		fmt.Println("unable to load system prompt file")
		return
	}

	SYSTEM_PROMPT = string(systemPromptFile)

	// setup discord session
	dg, err := discordgo.New("Bot " + DISCORD_TOKEN)
	if err != nil {
		fmt.Println("couldn't create discord session", err)
		return
	}

	// watch for new messages in bot channels
	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection", err)
		return
	}

	dg.UpdateWatchStatus(0, "for mentions")

	// create koboldcpp client
	Api, err = api.NewKoboldClient(API_URL)
	if err != nil {
		fmt.Println("unable to create koboldcpp client")
		return
	}

	fmt.Println("bot is running, ctrl-c to exit")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore if own message
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check message to see if bot was mentioned
	if !wasMentioned(s, m) || m.MentionEveryone {
		return
	}

	// TODO set discord typing indicator

	// store authors' usernames for stop strings later
	usernames := make([]string, 0)
	addUsername := func(username string) {
		formattedUsername := "\n" + username + ":"
		if slices.Index(usernames, formattedUsername) == -1 {
			usernames = append(usernames, formattedUsername)
		}
	}

	// grab the trigger message and the few most recent referenced messages
	messages := make([]*discordgo.Message, 0)
	messages = append(messages, m.Message)
	addUsername(m.Author.Username)
	previousMessage := m.ReferencedMessage
	for i := 0; i < 3; i++ {
		if previousMessage != nil {
			messages = append(messages, previousMessage)
			addUsername(previousMessage.Author.Username)
			previousMessage = previousMessage.ReferencedMessage
		} else {
			break
		}
	}

	// also grab the last 10 messages (deduplicated by id)
	contextMessages, err := s.ChannelMessages(m.ChannelID, 10, m.ID, "", "")
	if err != nil {
		fmt.Println("couldn't retrieve channel messages", err)
	} else {
		for _, message := range contextMessages {
			// skip already included messages
			if slices.IndexFunc(messages, func(ms *discordgo.Message) bool { return ms.ID == message.ID }) == -1 {
				messages = append(messages, message)
				addUsername(message.Author.Username)
			}
		}
	}

	// reverse the order of messages to be chronological
	slices.Reverse(messages)

	// build prompt w/ instructions, persona, and message history
	systemSeq := prompt.MISTRAL_V2.SystemStart
	systemEnd := prompt.MISTRAL_V2.SystemEnd
	userSeq := prompt.MISTRAL_V2.UserStart
	modelSeq := prompt.MISTRAL_V2.UserEnd + prompt.MISTRAL_V2.ModelStart

	// all the stop strings
	stopStrings := []string{prompt.MISTRAL_V2.ModelEnd, "\n\n\n", "\n---", "\n(Note:"}
	stopStrings = append(stopStrings, usernames...)

	var prompt strings.Builder

	prompt.WriteString(systemSeq + fmt.Sprintf(SYSTEM_PROMPT, PERSONA, PERSONA_DESC) + systemEnd)
	prompt.WriteString(userSeq + "## Conversation\n")

	for _, message := range messages {
		prompt.WriteString("\n[" + message.Timestamp.Format("2006-01-02 15:04:05") + "] " + message.Author.Username + ": " + trimSuffixes(message.ContentWithMentionsReplaced(), &stopStrings))
	}

	prompt.WriteString(modelSeq + PERSONA + ":")

	params := &api.KoboldParams{
		MaxContextLength: 16384,
		MaxLength:        250,
		Temperature:      0.5,
		DynaTempRange:    0,
		TopP:             1,
		MinP:             0.1,
		TopK:             0,
		TopA:             0,
		Typical:          1.0,
		Tfs:              1.0,
		RepPen:           1.0,
		RepPenRange:      1024,
		RepPenSlope:      0,
		SamplerOrder:     []int{6, 0, 1, 3, 4, 2, 5},
		SamplerSeed:      -1,
		StopSequence:     stopStrings,
		BanTokens:        false,
		TrimStop:         true,
		Prompt:           prompt.String(),
	}

	response, err := Api.Generate(params)
	if err != nil {
		fmt.Println("there was an error while waiting on a response from koboldcpp!", err)
	}

	if response.Status == "ok" {
		processedResponse := trimSuffixes(response.Text, &stopStrings)
		s.ChannelMessageSend(m.ChannelID, processedResponse)
	}

	// TODO save user and bot messages to user-specific memory file?

	// TODO generate custom mood status
}

func wasMentioned(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	mentioned := false
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			mentioned = true
			break
		}
	}
	if strings.Contains(strings.ToLower(m.Content), strings.ToLower(PERSONA)) {
		mentioned = true
	}
	return mentioned
}

func trimSuffixes(str string, suffixes *[]string) string {
	for _, suffix := range *suffixes {
		trimmedStr, trimmed := strings.CutSuffix(str, suffix)
		if trimmed {
			return trimmedStr
		}
	}
	return str
}
