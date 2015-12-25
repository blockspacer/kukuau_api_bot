package quests

import (
	"log"
	"math/rand"
	"time"

	s "msngr/structs"
	c "msngr/configuration"
	"msngr/db"

)

const (
	QUEST_STATE_KEY = "quest"
	SUBSCRIBED = "subscribed"
)

var subscribe_commands = []s.OutCommand{
	s.OutCommand{
		Title:    "Учавствовать",
		Action:   "subscribe",
		Position: 0,
		Repeated: false,
	},
}


var key_input_form = &s.OutForm{
	Title: "Форма ввода ключа для следующего задания",
	Type:  "form",
	Name:  "key_form",
	Text:  "Код: ?(code)",
	Fields: []s.OutField{
		s.OutField{
			Name: "key",
			Type: "text",
			Attributes: s.FieldAttribute{
				Label:    "Ваш найденый код",
				Required: true,
			},
		},
	},
}

var key_input_commands = []s.OutCommand{
	s.OutCommand{
		Title:    "Ввод найденного кода",
		Action:   "key_input",
		Position: 0,
		Repeated: false,
		Form:     key_input_form,
	},
}

type QuestCommandRequestProcessor struct {
	db.DbHandlerMixin
}

func (qcp *QuestCommandRequestProcessor) ProcessRequest(in *s.InPkg) *s.RequestResult {
	var result_commands []s.OutCommand
	if state, err := qcp.Users.GetUserMultiplyState(in.From, QUEST_STATE_KEY); err == nil && state == SUBSCRIBED {
		result_commands = key_input_commands
	} else {
		result_commands = subscribe_commands
	}
	result := s.RequestResult{Commands:&result_commands}
	return &result
}


type QuestSubscribeMessageProcessor struct {
	db.DbHandlerMixin
	AcceptPhrase   string
	RejectedPhrase string
	ErrorPhrase    string

}

func (qsmp *QuestSubscribeMessageProcessor) ProcessMessage(in *s.InPkg) *s.MessageResult {
	user, err := qsmp.Users.GetUserById(in.From)
	var text string
	if err != nil {
		text = qsmp.ErrorPhrase
		return &s.MessageResult{Commands:&subscribe_commands, Body:text, Type:"chat"}
	}
	if user != nil {
		if state, ok := user.GetStateValue(QUEST_STATE_KEY); ok && state == SUBSCRIBED {
			text = qsmp.RejectedPhrase
			return &s.MessageResult{Commands:&subscribe_commands, Body:text, Type:"chat"}
		} else {
			qsmp.Users.SetUserMultiplyState(in.From, QUEST_STATE_KEY, SUBSCRIBED)
			text = qsmp.AcceptPhrase
			return &s.MessageResult{Commands:&key_input_commands, Body:text, Type:"chat"}
		}
	} else {
		qsmp.Users.SetUserMultiplyState(in.From, QUEST_STATE_KEY, SUBSCRIBED)
		text = qsmp.AcceptPhrase
		return &s.MessageResult{Commands:&key_input_commands, Body:text, Type:"chat"}
	}
}

type QuestKeyInputMessageProcessor struct {
	db.DbHandlerMixin
}

func (qkimp QuestKeyInputMessageProcessor) ProcessMessage(in *s.InPkg) *s.MessageResult {
	var text string
	if state, err := qkimp.Users.GetUserMultiplyState(in.From, QUEST_STATE_KEY); err != nil || state != SUBSCRIBED {
		return &s.MessageResult{Commands:&subscribe_commands, Body:"Вы здесь быть не должны и делать это не можете.", Type:"chat"}
	}

	commands_ptr := in.Message.Commands
	if commands_ptr != nil {
		commands := *commands_ptr
		for _, command := range commands {
			if command.Action == "key_input" && command.Form.Name == "key_form" {
				for _, field := range command.Form.Fields {
					if field.Name == "key" {
						key := field.Data.Value
						log.Printf("QUESTS We have key from %v is: [%v]", in.From, key)
						r := rand.New(rand.NewSource(time.Now().UnixNano()))
						if r.Int31n(6) >= 3 {
							text = "Правильно! Ищите код там-то и сям-то. Вы почти что в шаге от 100500 тысяч миллионов денег."
						} else {
							text = "Не правильно, поищите код лучше."
						}
					}
				}
			}
		}
	}

	mr := s.MessageResult{Commands:&key_input_commands, Body:text, Type:"chat"}
	return &mr
}

func FormQuestBotContext(conf c.QuestConfig, db_handler *db.DbHandlerMixin) *s.BotContext {
	result := s.BotContext{}
	result.Request_commands = map[string]s.RequestCommandProcessor{
		"commands":&QuestCommandRequestProcessor{DbHandlerMixin:*db_handler},
	}
	result.Message_commands = map[string]s.MessageCommandProcessor{
		"subscribe":&QuestSubscribeMessageProcessor{DbHandlerMixin:*db_handler, AcceptPhrase:conf.AcceptPhrase, RejectedPhrase:conf.RejectPhrase, ErrorPhrase:conf.ErrorPhrase },
		"key_input":&QuestKeyInputMessageProcessor{DbHandlerMixin:*db_handler},
	}
	return &result

}