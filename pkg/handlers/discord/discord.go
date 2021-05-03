package discord

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/andersfylling/snowflake"
	"github.com/nickname32/discordhook"

	"github.com/bitnami-labs/kubewatch/config"
	"github.com/bitnami-labs/kubewatch/pkg/event"
)

var discordColors = map[string]int{
	"Normal":  0x7289DA,
	"Warning": 0xD4AF37,
	"Danger":  0xEF5350,
}

var DISCORD_HOOK = regexp.MustCompile(`https?://(?:(?:canary|ptb)\.)?discord(?:app)?.com/api/webhooks/(?P<id>\d+)/(?P<token>.*)`)

var discordErrMsg = `
%s

You need to set a Discord webhook using "--webhook/-w" or using environment variables:

export KW_DISCORD_WEBHOOK=discord_webhook

Command line flags will override environment variables

`

type Discord struct {
	ID    snowflake.Snowflake
	Token string
}

func (s *Discord) Init(c *config.Config) error {
	webhook := c.Handler.Discord.Webhook

	if webhook == "" {
		webhook = os.Getenv("KW_DISCORD_WEBHOOK")
	}

	match := DISCORD_HOOK.FindStringSubmatch(webhook)

	if match == nil {
		return fmt.Errorf(discordErrMsg, "Invalid Discord webhook URL")
	}

	result := make(map[string]string)

	for i, name := range DISCORD_HOOK.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	s.ID = snowflake.ParseSnowflakeString(result["id"])
	s.Token = result["token"]

	if s.Token == "" {
		return fmt.Errorf(discordErrMsg, "Invalid Discord webhook URL")
	}

	return nil
}

func (s *Discord) Handle(e event.Event) {
	hook, err := discordhook.NewWebhookAPI(s.ID, s.Token, false, nil)

	if err != nil {
		log.Fatalf(err.Error())
	}

	attachment := prepareDiscordAttachment(e, s)

	_, err = hook.Execute(context.TODO(), attachment, nil, "")

	if err != nil {
		log.Printf("%s\n", err)
		return
	}

	log.Printf("Message successfully sent to Discord hook %d", s.ID)
}

func prepareDiscordAttachment(e event.Event, s *Discord) *discordhook.WebhookExecuteParams {

	attachment := &discordhook.WebhookExecuteParams{
		Embeds: []*discordhook.Embed{
			{
				Title:       "Kubewatch",
				Description: e.Message(),
			},
		},
	}

	if color, ok := discordColors[e.Status]; ok {
		attachment.Embeds[0].Color = color
	}

	return attachment
}
