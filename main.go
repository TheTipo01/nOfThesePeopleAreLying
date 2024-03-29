package main

import (
	"fmt"
	"github.com/kkyr/fig"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Config holds data parsed from the config.yml
type Config struct {
	Token  string `fig:"cfg.Token" validate:"required"`
	Prefix string `fig:"cfg.Prefix" validate:"required"`
}

var (
	cfg   Config
	games = make(map[string]*game)
)

const (
	playEmoji = "🎮"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	err := fig.Load(&cfg, fig.File("config.yml"), fig.Dirs(".", "./data"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func main() {

	if cfg.Token == "" {
		fmt.Println("No cfg.Token provided. Please modify config.yml")
		return
	}

	if cfg.Prefix == "" {
		fmt.Println("No cfg.Prefix provided. Please modify config.yml")
		return
	}

	// Create a new Discord session using the provided bot cfg.Token.
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(reactionAdd)
	dg.AddHandler(reactionRemove)
	dg.AddHandler(ready)

	// Intents for getting the correct event from discord
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsDirectMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

func ready(s *discordgo.Session, _ *discordgo.Ready) {

	// Set the playing status.
	err := s.UpdateGameStatus(0, cfg.Prefix+"help")
	if err != nil {
		fmt.Println("Can't set status,", err)
	}
}

// This function will be called (due to AddHandler above) every time a new
// m is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself and all the messages from bots
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	// Create new game
	if m.Content == cfg.Prefix+"play" && games[m.GuildID] == nil {
		sntM, err := s.ChannelMessageSend(m.ChannelID, "To play, add a reaction!")
		if err != nil {
			fmt.Println(err)
			return
		}

		err = s.MessageReactionAdd(m.ChannelID, sntM.ID, playEmoji)
		if err != nil {
			fmt.Println(err)
			_ = s.ChannelMessageDelete(m.ChannelID, sntM.ID)
			return
		}

		games[m.GuildID] = &game{sntM, nil, false, "", 0, m.ChannelID, m.GuildID, false, "", nil, ""}

		return
	}

	// Starts the game
	if m.Content == cfg.Prefix+"start" && games[m.GuildID] != nil {

		_ = s.ChannelMessageDelete(games[m.GuildID].m.ChannelID, games[m.GuildID].m.ID)
		games[m.GuildID].m = nil

		games[m.GuildID].round++

		gamer := getRand(m.GuildID, false)
		games[m.GuildID].guesser = gamer.id

		mex, _ := s.ChannelMessageSend(m.ChannelID, "Round "+strconv.Itoa(games[m.GuildID].round)+" started!\n"+"<@"+gamer.id+"> needs to guess! to guess!\nSend your article in private to me!")

		// Add the message, to delete it later
		games[m.GuildID].messages = append(games[m.GuildID].messages, mex)

		return
	}

	// Remove player from the game
	if m.Content == cfg.Prefix+"remove" && games[m.GuildID] != nil {
		games[m.GuildID].players[m.ID] = nil
		mex, _ := s.ChannelMessageSend(m.GuildID, "You have been removed from the game!")
		// Add the message, to delete it later
		games[m.GuildID].messages = append(games[m.GuildID].messages, mex)

		return
	}

	// Stop the game
	if m.Content == cfg.Prefix+"stop" && games[m.GuildID] != nil {
		games[m.GuildID] = nil
		mex, _ := s.ChannelMessageSend(m.ChannelID, "Game has been stopped!")

		time.Sleep(time.Second)
		_ = s.ChannelMessageDelete(mex.ChannelID, mex.ID)

		return
	}

	// Help message
	if m.Content == cfg.Prefix+"help" {
		mex, _ := s.ChannelMessageSend(m.ChannelID, "```"+cfg.Prefix+"play - The bot sends a message to start the session\n"+cfg.Prefix+"start - Start the session\n"+cfg.Prefix+"remove - Removes yourself from the game\n"+cfg.Prefix+"stop - Stop the current game```")

		time.Sleep(5 * time.Second)
		_ = s.ChannelMessageDelete(mex.ChannelID, mex.ID)

		return
	}

	// Private messages, for doing all sorts of things
	if m.GuildID == "" {
		guild := getGuildFromUser(m.Author.ID)
		if guild != "" {
			// Check if the guesser guessed something
			if games[guild].response && games[guild].guesser == m.Author.ID {
				games[guild].response = false

				// Remove old messages
				removeMessages(s, games[guild].messages)
				games[guild].messages = nil

				// Saves previous guesser
				games[guild].previousGuesser = games[guild].guesser

				if didYoUGuess(guild, m.Content) {
					updatePoint(guild, "")
					mex, _ := s.ChannelMessageSend(games[guild].channel, "Correct!\nUpdated leaderboard: \n"+leaderboard(guild))
					// Add the message, to delete it later
					games[guild].messages = append(games[guild].messages, mex)
				} else {
					updatePoint(guild, searchUser(m.Content))
					mex, _ := s.ChannelMessageSend(games[guild].channel, "Wrong! The correct user was "+games[guild].players[games[guild].chosenOne].username+"!\nUpdated leaderboard: \n"+leaderboard(guild))
					// Add the message, to delete it later
					games[guild].messages = append(games[guild].messages, mex)
				}

				// New round
				games[guild].round++

				gamer := getRand(guild, false)
				games[guild].guesser = gamer.id

				mex, _ := s.ChannelMessageSend(games[guild].channel, "Round "+strconv.Itoa(games[guild].round)+" started!\n"+"<@"+gamer.id+"> needs to guess!\nSend your article in private to me!")
				// Add the message, to delete it later
				games[guild].messages = append(games[guild].messages, mex)

				return
			}

			if games[guild].guesser == m.Author.ID {
				_, _ = s.ChannelMessageSend(m.ChannelID, "You need to guess!\nWait for your friends to finish sending articles in!")
				return
			}

			games[guild].players[m.Author.ID].article = m.Content
			_, _ = s.ChannelMessageSend(m.ChannelID, "Got your article!")

			if haveWeFinished(guild) {
				random := getRand(guild, true)
				games[guild].chosenOne = random.id

				games[guild].response = true
				mex, _ := s.ChannelMessageSend(games[guild].channel, "All articles are in!\nThe selected one is: "+random.article+"\nAnswer in private with only the username!")
				// Add the message, to delete it later
				games[guild].messages = append(games[guild].messages, mex)
			}
		}

		return
	}
}

func reactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {

	// Ignore all messages created by the bot itself
	if r.UserID == s.State.User.ID {
		return
	}

	if games[r.GuildID] != nil && !games[r.GuildID].started && games[r.GuildID].m != nil && r.MessageID == games[r.GuildID].m.ID && r.Emoji.Name == playEmoji {
		u, err := s.GuildMember(r.GuildID, r.UserID)
		if err != nil {
			fmt.Println(err)
			return
		}

		if games[r.GuildID].players == nil {
			games[r.GuildID].players = make(map[string]*gamer)
		}

		games[r.GuildID].players[r.UserID] = &gamer{r.UserID, 0, u.User.Username, ""}
	}

}

func reactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {

	// Ignore all messages created by the bot itself
	if r.UserID == s.State.User.ID {
		return
	}

	if games[r.GuildID] != nil && !games[r.GuildID].started && r.MessageID == games[r.GuildID].m.ID && r.Emoji.Name == playEmoji {
		games[r.GuildID].players[r.UserID] = nil
	}

}
