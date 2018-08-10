package autorole

import (
	"github.com/jonas747/discordgo"
	"github.com/jonas747/yagpdb/common"
	"github.com/jonas747/yagpdb/common/pubsub"
	"github.com/jonas747/yagpdb/web"
	"github.com/mediocregopher/radix.v3"
	"goji.io"
	"goji.io/pat"
	"html/template"
	"net/http"
)

type Form struct {
	GeneralConfig `valid:"traverse"`
}

var _ web.SimpleConfigSaver = (*Form)(nil)

func (f Form) Save(guildID int64) error {
	pubsub.Publish("autorole_stop_processing", guildID, nil)

	err := common.SetRedisJson(KeyGeneral(guildID), f.GeneralConfig)
	if err != nil {
		return err
	}

	return nil
}

func (f Form) Name() string {
	return "Autorole"
}

func (p *Plugin) InitWeb() {
	tmplPathSettings := "templates/plugins/autorole.html"
	if common.Testing {
		tmplPathSettings = "../../autorole/assets/autorole.html"
	}

	web.Templates = template.Must(web.Templates.ParseFiles(tmplPathSettings))

	muxer := goji.SubMux()

	web.CPMux.Handle(pat.New("/autorole"), muxer)
	web.CPMux.Handle(pat.New("/autorole/*"), muxer)

	muxer.Use(web.RequireFullGuildMW) // need roles
	muxer.Use(web.RequireBotMemberMW) // need the bot's role
	muxer.Use(web.RequirePermMW(discordgo.PermissionManageRoles))

	getHandler := web.RenderHandler(HandleAutoroles, "cp_autorole")

	muxer.Handle(pat.Get(""), getHandler)
	muxer.Handle(pat.Get("/"), getHandler)

	muxer.Handle(pat.Post(""), web.SimpleConfigSaverHandler(Form{}, getHandler))
	muxer.Handle(pat.Post("/"), web.SimpleConfigSaverHandler(Form{}, getHandler))
}

func HandleAutoroles(w http.ResponseWriter, r *http.Request) interface{} {
	ctx := r.Context()
	activeGuild, tmpl := web.GetBaseCPContextData(ctx)

	general, err := GetGeneralConfig(activeGuild.ID)
	web.CheckErr(tmpl, err, "Failed retrieving general config (contact support)", web.CtxLogger(r.Context()).Error)
	tmpl["Autorole"] = general

	var proc int
	common.RedisPool.Do(radix.Cmd(&proc, "GET", KeyProcessing(activeGuild.ID)))
	tmpl["Processing"] = proc
	tmpl["ProcessingETA"] = int(proc / 60)

	return tmpl

}
