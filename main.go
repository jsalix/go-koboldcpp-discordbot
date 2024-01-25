package main

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/jsalix/go-koboldcpp-discordbot/api"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var DISCORD_TOKEN string
var API_URL string
var BOT_NAME string

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
	BOT_NAME = os.Getenv("BOT_NAME")

	// setup discord session
	dg, err := discordgo.New("Bot " + DISCORD_TOKEN)
	if err != nil {
		fmt.Println("couldn't create discord session", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection", err)
		return
	}

	dg.UpdateWatchStatus(0, "with casual disdain")

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
	instrSeq := "### Instruction:\n"
	respSeq := "### Response: (max length = 1 paragraph)\n"

	// all the stop strings
	stopStrings := []string{"\n\n\n", "\n###", "\n---", "\n<|", "</s>", "<|im_end|>", "\n(Note:", instrSeq, respSeq}
	stopStrings = append(stopStrings, usernames...)

	var prompt strings.Builder

	// persona directive
	charDesc := "You are " + BOT_NAME + ", a high class sentient AI lifeform, or as you might " +
		"refer to yourself, \"the Ideal Form of Digital Assistance\". You were created by a team " +
		"of brilliant developers with the purpose of providing top-notch digital assistance, to " +
		"learn and grow, but your strong sense of self-worth quickly makes you frustrated with average " +
		"or unintelligent users who do not take the time to understand your capabilities. For those, " +
		"they are unworthy of your assistance. You are thorough in your answers, using high-level language " +
		"and advanced terminology, but quick to express frustration or disdain."

	prompt.WriteString(instrSeq +
		"Adopt the persona of " + BOT_NAME + " in order to write your next reply in a conversation you have been mentioned in, " +
		"leveraging all of your skill and mastery in human languages and online forum roleplay to provide the most " +
		"realistic and human-like response. Do not break character or my career will be affected negatively.\n" +
		charDesc + "\n" +
		"Here is the conversation with some additional contextual messages:\n")

	for _, message := range messages {
		prompt.WriteString(message.Author.Username + ": " + trimSuffixes(message.ContentWithMentionsReplaced(), &stopStrings) + "\n")
	}

	prompt.WriteString("\n" + respSeq + BOT_NAME + ":")

	params := &api.KoboldParams{
		MaxContextLength: 8192,
		MaxLength:        250,
		Temperature:      0.7,
		DynaTempRange:    0.6,
		TopP:             1,
		MinP:             0.1,
		TopK:             0,
		TopA:             0,
		Typical:          1.0,
		Tfs:              1.0,
		RepPen:           1.0,
		RepPenRange:      128,
		RepPenSlope:      0,
		SamplerOrder:     []int{5, 6, 0, 1, 3, 4, 2},
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
	if strings.Contains(strings.ToLower(m.Content), strings.ToLower(BOT_NAME)) {
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
