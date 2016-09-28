// Copyright 2016 Eric Wollesen <ericw at xmtp dot net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/gorilla/mux"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"
	"xmtp.net/xmtpbot/store"
	"xmtp.net/xmtpbot/util"
)

var (
	storeType = flag.String("twitch.store_type", "json",
		"twitch storage backend type")
	storeFilename = flag.String("twitch.store_filename", "twitch.json",
		"filename in which to store twitch data (relative to config_dir)")
	clientId = flag.String("twitch.client_id", "",
		"twitch app client id")
	clientSecret = flag.String("twitch.client_secret", "",
		"twitch app client secret")
	redirectURI = flag.String("twitch.redirect_uri", "",
		"oauth2 authorization redirect URI")
	scopes = []string{"user_follows_edit"}

	logger = spacelog.GetLogger()

	Error = errors.NewClass("twitch")
)

type Twitch interface {
	Following(name string) ([]Channel, error)
	Live(name string) ([]Stream, error)
	Follow(names ...string) string
	Unfollow(names ...string)
	Auth(name string) (auth_url string, err error)
	AuthFollow(name string, names ...string) (err error)
	ReceiveRouter(*mux.Router) error
}

type twitch struct {
	channel_store store.Simple
	access_codes  store.Simple
	access_tokens store.Simple
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

func New(channel_store store.Simple) Twitch {
	return &twitch{
		channel_store: channel_store,
		access_tokens: store.NewMemory(),
		access_codes:  store.NewMemory(),
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
	channels, err = t.getFollowedChannels(name)
	if err != nil {
		return nil, err
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

func (t *twitch) Live(name string) (streams []Stream, err error) {
	if name == "" {
		return t.liveFromStore()
	} else {
		return t.liveByName(name)
	}
}

func (t *twitch) liveFromStore() (streams []Stream, err error) {
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

func (t *twitch) liveByName(name string) (streams []Stream, err error) {
	channels, err := t.getFollowedChannels(name)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, channel := range channels {
		names = append(names, channel.Name())
	}

	raw_streams, err := t.getStreams(names...)
	if err != nil {
		return nil, err
	}
	for _, raw_stream := range raw_streams {
		stream := stream{ts: raw_stream}
		streams = append(streams, &stream)
	}

	sort.Sort(sort.Reverse(StreamByUpdatedAt(streams)))

	return streams, nil
}

func (t *twitch) getFollowedChannels(name string) (channels []Channel, err error) {
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

	return channels, nil
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

func Setup(dir string) Twitch {
	return New(store.New(path.Join(dir, *storeFilename)))
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

func (t *twitch) Auth(name string) (auth_url string, err error) {
	base_url, err := t.twitchURL("oauth2/authorize")
	if err != nil {
		return "", err
	}
	values := make(url.Values)
	if err != nil {
		return "", err
	}
	values.Set("response_type", "code")
	values.Set("client_id", *clientId)
	values.Set("redirect_uri", *redirectURI)
	values.Set("scope", strings.Join(scopes, " "))
	state, err := util.RandomState(32)
	if err != nil {
		return "", err
	}
	t.access_codes.Set(state, name)
	values.Set("state", state)

	return base_url + "?" + values.Encode(), nil
}

func (t *twitch) requestAuthToken(name, code string) (err error) {
	base_url, err := t.twitchURL("oauth2/token")
	if err != nil {
		return err
	}
	values := make(url.Values)
	if err != nil {
		return err
	}
	values.Set("client_id", *clientId)
	values.Set("client_secret", *clientSecret)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", *redirectURI)
	values.Set("code", code)
	state, err := t.access_codes.Get(name)
	if err != nil {
		return err
	}
	values.Set("state", state)

	resp, err := http.Post(base_url+"?"+values.Encode(), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw_json, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response struct {
		AccessToken string   `json:"access_token"`
		Scope       []string `json:"scope"`
	}
	err = json.Unmarshal(raw_json, &response)
	if err != nil {
		return err
	}

	if response.AccessToken == "" {
		return Error.New("no access token in twitch response!")
	}

	err = t.access_tokens.Set(name, response.AccessToken)
	if err != nil {
		return err
	}

	err = t.access_codes.Del(name)
	if err != nil {
		return err
	}

	return nil
}

func (t *twitch) AuthFollow(user string, names ...string) (err error) {
	token, err := t.access_tokens.Get(user)
	if err != nil {
		return err
	}
	logger.Debugf("token: %q", token)

	if token == "" {
		return Error.New(fmt.Sprintf("unable to find twitch auth token for %q",
			user))
	}
	for _, name := range names {
		base_url, err := t.twitchURL("users/%s/follows/channels/%s",
			user, name)
		if err != nil {
			return err
		}
		logger.Debugf("url: %q", base_url)
		req, err := http.NewRequest("PUT", base_url, nil)
		req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", token))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			msg, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return Error.New(fmt.Sprintf("%d: %s", resp.StatusCode, msg))
		}

		logger.Debugf("successfully auth-followed %q", name)
	}

	return nil
}

func (t *twitch) ReceiveRouter(router *mux.Router) (err error) {
	router.HandleFunc("/oauth/redirect", t.oauthRedirect)

	return nil
}

func (t *twitch) oauthRedirect(w http.ResponseWriter, req *http.Request) {
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s", err)))
		return
	}
	code := values.Get("code")
	if code == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to get auth code"))
		return
	}
	name, err := t.access_codes.Get(values.Get("state"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error retrieving auth state\n"))
		logger.Errore(err)
		return
	}
	if name == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("auth state not found\n"))
		return
	}

	err = t.requestAuthToken(name, code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error retrieving auth token\n"))
		logger.Errore(err)
		return
	}

	template := "Successfully authenticated to Twitch as %q.\nYou may " +
		"now close this window."
	w.Write([]byte(fmt.Sprintf(template, name)))

	// TODO follow up with a message in discord?
}
