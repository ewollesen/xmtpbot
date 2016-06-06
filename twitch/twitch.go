// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package twitch

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"xmtp.net/xmtpbot/config"
	"xmtp.net/xmtpbot/store"

	"github.com/spacemonkeygo/spacelog"
)

var (
	storeType = flag.String("twitch.store_type", "json",
		"twitch storage backend type")
	storeFilename = flag.String("twitch.store_filename",
		path.Join(*config.Dir, "twitch.json"),
		"filename in which to store twitch data")

	logger = spacelog.GetLogger()
)

type Twitch interface {
	Following(name string) ([]Channel, error)
	Live() ([]Stream, error)
	Follow(names ...string) string
	Unfollow(names ...string)
}

type twitch struct {
	channel_store store.Simple
}

type Channel interface {
	Name() string
	URL() string
	UpdatedAt() time.Time
}

type twitchChannel struct {
	Mature                       bool      `json:"mature"`
	Status                       string    `json:"status"`
	BroadcasterLanguage          string    `json:"broadcaster_language"`
	DisplayName                  string    `json:"display_name"`
	Game                         string    `json:"game"`
	Delay                        int       `json:"delay"`
	Language                     string    `json:"language"`
	Id                           int       `json:"_id"`
	Name                         string    `json:"name"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
	Logo                         string    `json:"logo"`
	Banner                       string    `json:"banner"`
	VideoBanner                  string    `json:"video_banner"`
	Background                   string    `json:"background"`
	ProfileBanner                string    `json:"profile_banner"`
	ProfileBannerBackgroundColor string    `json:"profile_banner_background_color"`
	Partner                      bool      `json:"partner"`
	URL                          string    `json:"url"`
	Views                        int       `json:"views"`
	Followers                    int       `json:"followers"`
	Links                        *apiLinks `json:"_links,omitempty"`
}

type channel struct {
	tc *twitchChannel
}

type Stream interface {
	Name() string
	URL() string
	Live() bool
	UpdatedAt() time.Time
}

type twitchStream struct {
	Game        string         `json:"game"`
	Viewers     int            `json:"viewers"`
	AverageFPS  float32        `json:"average_fps"`
	Delay       int            `json:"delay"`
	VideoHeight int            `json:"video_height"`
	IsPlaylist  bool           `json:"is_playlist"`
	CreatedAt   time.Time      `json:"created_at"`
	Id          int            `json:"_id"`
	Channel     *twitchChannel `json:"channel"`
}

type stream struct {
	ts *twitchStream
}

type streamResponse struct {
	Streams []*twitchStream `json:"streams"`
}

type apiLinks struct {
	Self          string `json:"self,omitempty"`
	Follows       string `json:"follows,omitempty"`
	Commercial    string `json:"commercial,omitempty"`
	StreamKey     string `json:"stream_key,omitempty"`
	Chat          string `json:"chat,omitempty"`
	Features      string `json:"features,omitempty"`
	Subscriptions string `json:"subscriptions,omitempty"`
	Editors       string `json:"editors,omitempty"`
	Teams         string `json:"teams,omitempty"`
	Videos        string `json:"videos,omitempty"`
	Channel       string `json:"channel,omitempty"`
}

func New(store store.Simple) Twitch {
	return &twitch{
		channel_store: store,
	}
}

func (t *twitch) Following(name string) (channels []Channel, err error) {
	if name == "" {
		return t.followingFromStore()
	} else {
		return t.followingByName(name)
	}
}

func (t *twitch) followingFromStore() (channels []Channel, err error) {
	t.channel_store.Iterate(func(key, value string) {
		if value == "" {
			logger.Debugf("retrieving channel info for %q", key)
			tc, err := t.getChannel(key)
			if err != nil {
				return
			}
			channels = append(channels, &channel{tc: tc})
		} else {
			logger.Debugf("using cached channel info for %q", key)
			channels = append(channels, &channel{tc: &twitchChannel{
				DisplayName: key,
				URL:         value,
			}})
		}
	})
	sort.Sort(ChannelByName(channels))

	return channels, nil
}

func (t *twitch) followingByName(name string) (channels []Channel, err error) {
	url, err := t.twitchURL("users/%s/follows/channels", name)
	if err != nil {
		return nil, err
	}
	// TODO support more than 100 links
	url += "?limit=100"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw_json, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Debugf("raw json response: %s", string(raw_json))
		return nil, err
	}

	var response_structure struct {
		Follows []struct {
			Channel *twitchChannel `json:"channel"`
		} `json:"follows"`
	}
	err = json.Unmarshal(raw_json, &response_structure)
	if err != nil {
		logger.Debugf("raw json response: %s", string(raw_json))
		return nil, err
	}

	for _, resp := range response_structure.Follows {
		channels = append(channels, &channel{tc: resp.Channel})
	}

	sort.Sort(ChannelByName(channels))

	return channels, nil
}

func (t *twitch) getChannel(name string) (tc *twitchChannel, err error) {
	url, err := t.twitchURL("channels/%s", name)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw_json, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw_json, &tc)
	if err != nil {
		logger.Debugf("raw json response: %s", string(raw_json))
		return nil, err
	}

	return tc, nil
}

func (t *twitch) getStreams(names ...string) ([]*twitchStream, error) {
	url_str, err := t.twitchURL("streams")
	if err != nil {
		return nil, err
	}

	values := make(url.Values)
	if err != nil {
		return nil, err
	}
	values.Set("channel", strings.Join(names, ","))
	url_str += "?" + values.Encode()

	resp, err := http.Get(url_str)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw_json, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response streamResponse
	err = json.Unmarshal(raw_json, &response)
	if err != nil {
		buf := bytes.NewBuffer([]byte{})
		json.Indent(buf, raw_json, "", "  ")
		logger.Debugf("failed unmarshalling: %s", string(buf.Bytes()))
		return nil, err
	}

	return response.Streams, nil
}

func (t *twitch) twitchURL(path_template string, args ...interface{}) (
	twitch_url string, err error) {

	url, err := url.Parse("https://api.twitch.tv")
	if err != nil {
		return "", err
	}
	url.Path = fmt.Sprintf(path.Join("/kraken", path_template), args...)
	return url.String(), nil
}

func (t *twitch) Live() (streams []Stream, err error) {
	var stream_names []string
	t.channel_store.Iterate(func(key, value string) {
		stream_names = append(stream_names, key)
	})

	xs, err := t.getStreams(stream_names...)
	if err != nil {
		return
	}
	for _, x := range xs {
		s := stream{ts: x}
		streams = append(streams, &s)
	}

	sort.Sort(sort.Reverse(StreamByUpdatedAt(streams)))

	return streams, nil
}

func (c *channel) Name() string {
	return c.tc.DisplayName
}

func (c *channel) URL() string {
	return c.tc.URL
}

func (c *channel) UpdatedAt() time.Time {
	return c.tc.UpdatedAt
}

func (s *stream) Name() string {
	return s.ts.Channel.DisplayName
}

func (s *stream) URL() string {
	return s.ts.Channel.URL
}

func (s *stream) Live() bool {
	return s.ts != nil
}

func (s *stream) UpdatedAt() time.Time {
	return s.ts.Channel.UpdatedAt
}

func (t *twitch) Unfollow(names ...string) {
	for _, name := range names {
		t.channel_store.Del(name)
	}
}

func (t *twitch) Follow(names ...string) string {
	for _, name := range names {
		ch, err := t.getChannel(name)
		if err != nil {
			return fmt.Sprintf("couldn't retrieve channel %q", name)
		}

		t.channel_store.Set(name, ch.URL)
	}
	return "OK"
}

func Setup() Twitch {
	return New(store.New(*storeFilename))
}

type ChannelByName []Channel

func (a ChannelByName) Len() int      { return len(a) }
func (a ChannelByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ChannelByName) Less(i, j int) bool {
	left := strings.ToLower(a[i].Name())
	right := strings.ToLower(a[j].Name())
	return strings.Compare(left, right) < 0
}

type StreamByUpdatedAt []Stream

func (a StreamByUpdatedAt) Len() int           { return len(a) }
func (a StreamByUpdatedAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StreamByUpdatedAt) Less(i, j int) bool { return a[i].UpdatedAt().Before(a[j].UpdatedAt()) }
