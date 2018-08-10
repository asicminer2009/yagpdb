package mqueue

import (
	"encoding/json"
	"github.com/jonas747/discordgo"
	"github.com/jonas747/yagpdb/bot"
	"github.com/jonas747/yagpdb/common"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-kallax.v1"
	"strconv"
	"sync"
	"time"
)

var (
	sources  = make(map[string]PluginWithErrorHandler)
	stopChan = make(chan *sync.WaitGroup)

	currentlyProcessing     = make([]int64, 0)
	currentlyProcessingLock sync.RWMutex

	store *QueuedElementStore

	startedLock sync.Mutex
	started     bool
)

type PluginWithErrorHandler interface {
	HandleMQueueError(elem *QueuedElement, err error)
}

var (
	_ bot.BotInitHandler    = (*Plugin)(nil)
	_ bot.BotStopperHandler = (*Plugin)(nil)
)

type Plugin struct {
}

func (p *Plugin) Name() string {
	return "mqueue"
}

func RegisterPlugin() {
	p := &Plugin{}
	common.RegisterPlugin(p)
}

func InitStores() {
	// Init table
	_, err := common.PQ.Exec(`CREATE TABLE IF NOT EXISTS mqueue (
	id serial NOT NULL PRIMARY KEY,
	source text NOT NULL,
	source_id text NOT NULL,
	message_str text NOT NULL,
	message_embed text NOT NULL,
	channel text NOT NULL,
	processed boolean NOT NULL
);

CREATE INDEX IF NOT EXISTS mqueue_processed_x ON mqueue(processed);
`)
	if err != nil {
		panic("mqueue: " + err.Error())
	}

	store = NewQueuedElementStore(common.PQ)
}

func RegisterSource(name string, source PluginWithErrorHandler) {
	sources[name] = source
}

func QueueMessageString(source, sourceID, channel, message string) {
	elem := &QueuedElement{
		Source:     source,
		SourceID:   sourceID,
		Channel:    channel,
		MessageStr: message,
	}

	store.Insert(elem)
}

func QueueMessageEmbed(source, sourceID, channel string, embed *discordgo.MessageEmbed) {
	encoded, err := json.Marshal(embed)
	if err != nil {
		logrus.WithError(err).Error("MQueue: Failed encoding message")
		return
	}

	elem := &QueuedElement{
		Source:       source,
		SourceID:     sourceID,
		Channel:      channel,
		MessageEmbed: string(encoded),
	}

	store.Insert(elem)
}

func (p *Plugin) BotInit() {
	go startPolling()
}

func (p *Plugin) StopBot(wg *sync.WaitGroup) {
	startedLock.Lock()
	if !started {
		startedLock.Unlock()
		wg.Done()
		return
	}
	startedLock.Unlock()
	stopChan <- wg
}

func startPolling() {
	startedLock.Lock()
	if started {
		startedLock.Unlock()
		panic("Already started mqueue")
	}
	started = true
	startedLock.Unlock()

	ticker := time.NewTicker(time.Second)
	tickerClean := time.NewTicker(time.Hour)
	for {
		select {
		case wg := <-stopChan:
			shutdown(wg)
			return
		case <-ticker.C:
			poll()
		case <-tickerClean.C:
			go func() {
				result, err := common.PQ.Exec("DELETE FROM mqueue WHERE processed=true")
				if err != nil {
					logrus.WithError(err).Error("Failed cleaning mqueue db")
				} else {
					rows, err := result.RowsAffected()
					if err == nil {
						logrus.Println("mqueue cleaned up ", rows, " rows")
					}
				}
			}()
		}
	}
}

func shutdown(wg *sync.WaitGroup) {
	for i := 0; i < 10; i++ {
		currentlyProcessingLock.RLock()
		num := len(currentlyProcessing)
		currentlyProcessingLock.RUnlock()
		if num < 1 {
			break
		}
		time.Sleep(time.Second)
	}
	wg.Done()
}

func poll() {

	currentlyProcessingLock.RLock()
	processing := make([]int64, len(currentlyProcessing))
	copy(processing, currentlyProcessing)
	currentlyProcessingLock.RUnlock()

	elems, err := store.FindAll(NewQueuedElementQuery().Where(kallax.Eq(Schema.QueuedElement.Processed, false)))
	if err != nil {
		logrus.WithError(err).Error("MQueue: Failed polling message queue")
		return
	}

	currentlyProcessingLock.Lock()
OUTER:
	for _, v := range elems {
		for _, current := range processing {
			if v.ID == current {
				continue OUTER
			}
		}
		currentlyProcessing = append(currentlyProcessing, v.ID)
		go process(v)
	}
	currentlyProcessingLock.Unlock()
}

func process(elem *QueuedElement) {
	queueLogger := logrus.WithField("mq_id", elem.ID)

	defer func() {
		elem.Processed = true
		_, err := store.Save(elem)
		if err != nil {
			queueLogger.WithError(err).Error("MQueue: Failed marking elem as processed")
		}

		currentlyProcessingLock.Lock()
		for i, v := range currentlyProcessing {
			if v == elem.ID {
				currentlyProcessing = append(currentlyProcessing[:i], currentlyProcessing[i+1:]...)
				break
			}
		}
		currentlyProcessingLock.Unlock()
	}()

	var embed *discordgo.MessageEmbed
	if len(elem.MessageEmbed) > 0 {
		err := json.Unmarshal([]byte(elem.MessageEmbed), &embed)
		if err != nil {
			queueLogger.Error("MQueue: Failed decoding message embed")
		}
	}

	parsedChannel, err := strconv.ParseInt(elem.Channel, 10, 64)
	if err != nil {
		queueLogger.WithError(err).Error("Failed parsing Channel")
	}
	for {
		var err error
		if elem.MessageStr != "" {
			_, err = common.BotSession.ChannelMessageSend(parsedChannel, elem.MessageStr)
		} else if embed != nil {
			_, err = common.BotSession.ChannelMessageSendEmbed(parsedChannel, embed)
		} else {
			queueLogger.Error("MQueue: Both MessageEmbed and MessageStr empty")
			break
		}

		if err == nil {
			break
		}

		if e, ok := err.(*discordgo.RESTError); ok {
			if (e.Response != nil && e.Response.StatusCode >= 400 && e.Response.StatusCode < 500) || (e.Message != nil && e.Message.Code != 0) {
				if source, ok := sources[elem.Source]; ok {
					source.HandleMQueueError(elem, err)
				}
				break
			}
		}

		queueLogger.Warn("MQueue: Non-discord related error when sending message, retrying. ", err)
		time.Sleep(time.Second)
	}
}
