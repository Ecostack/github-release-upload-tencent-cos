package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func getHttpClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // Connect timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 5 * time.Second, // Read timeout
	}
	client := &http.Client{
		Timeout:   60 * time.Second, // Overall request timeout
		Transport: transport,
	}

	return client
}

func downloadFile(url, filename, token string) error {
	client := getHttpClient()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/octet-stream")

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("response.status is not 200 - " + response.Status)
	}
	defer response.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	log.Printf("%s downloaded successfully.\n", filename)
	return nil
}

func downloadFileFromGithub(owner, repo, id, token, filename string) error {
	urlStr := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/assets/%s", owner, repo, id)
	return downloadFile(urlStr, filename, token)
}

type ResponseBody struct {
	Url       string `json:"url"`
	AssetsUrl string `json:"assets_url"`
	UploadUrl string `json:"upload_url"`
	HtmlUrl   string `json:"html_url"`
	Id        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		Id                int    `json:"id"`
		NodeId            string `json:"node_id"`
		AvatarUrl         string `json:"avatar_url"`
		GravatarId        string `json:"gravatar_id"`
		Url               string `json:"url"`
		HtmlUrl           string `json:"html_url"`
		FollowersUrl      string `json:"followers_url"`
		FollowingUrl      string `json:"following_url"`
		GistsUrl          string `json:"gists_url"`
		StarredUrl        string `json:"starred_url"`
		SubscriptionsUrl  string `json:"subscriptions_url"`
		OrganizationsUrl  string `json:"organizations_url"`
		ReposUrl          string `json:"repos_url"`
		EventsUrl         string `json:"events_url"`
		ReceivedEventsUrl string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeId          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []struct {
		Url      string `json:"url"`
		Id       int    `json:"id"`
		NodeId   string `json:"node_id"`
		Name     string `json:"name"`
		Label    string `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			Id                int    `json:"id"`
			NodeId            string `json:"node_id"`
			AvatarUrl         string `json:"avatar_url"`
			GravatarId        string `json:"gravatar_id"`
			Url               string `json:"url"`
			HtmlUrl           string `json:"html_url"`
			FollowersUrl      string `json:"followers_url"`
			FollowingUrl      string `json:"following_url"`
			GistsUrl          string `json:"gists_url"`
			StarredUrl        string `json:"starred_url"`
			SubscriptionsUrl  string `json:"subscriptions_url"`
			OrganizationsUrl  string `json:"organizations_url"`
			ReposUrl          string `json:"repos_url"`
			EventsUrl         string `json:"events_url"`
			ReceivedEventsUrl string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadUrl string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballUrl string `json:"tarball_url"`
	ZipballUrl string `json:"zipball_url"`
	Body       string `json:"body"`
}

func downloadLatestAssets(owner, repo, token string) error {
	client := getHttpClient()

	// Get the latest release information
	urlStr := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to retrieve release information for %s. Error: %s", repo, response.Status)
	}

	var releaseData ResponseBody

	b, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(b, &releaseData)
	if err != nil {
		return err
	}

	log.Printf("Downloading assets for %s %s:\n", repo, releaseData.TagName)

	// Download each asset
	for _, asset := range releaseData.Assets {
		assetName := asset.Name

		log.Printf("Checking existing %s...\n", assetName)

		exists, err := hasFileAtTencentCOS(assetName)
		if err != nil {
			log.Fatalf("Failed to check %s. Error: %s\n", assetName, err.Error())
		}
		if exists {
			log.Println("File already exists, skip download.")
			continue
		}

		log.Printf("Downloading %s...\n", assetName)

		err = downloadFileFromGithub(owner, repo, strconv.FormatInt(int64(asset.Id), 10), token, assetName)
		if err != nil {
			log.Fatalf("Failed to download %s. Error: %s\n", assetName, err.Error())
		}
		err = uploadFileToTencentCOS(assetName)
		if err != nil {
			log.Fatalf("Failed to upload %s. Error: %s\n", assetName, err.Error())
		}

		newUrl := getTencentCOSURL(assetName)
		sendMessageInDiscord("New Release :> " + newUrl)

		err = os.Remove(assetName)
		if err != nil {
			log.Fatalln("Error deleting file:", err)
		}
	}

	return nil
}

var cosClient *cos.Client

func initCOS() {
	// Set your Tencent Cloud COS API credentials
	secretId := os.Getenv("COS_SECRET_ID")
	secretKey := os.Getenv("COS_SECRET_KEY")

	if secretId == "" || secretKey == "" {
		log.Fatal("COS_SECRET_ID or COS_SECRET_KEY is empty")
	}

	urlStr := getTencentBucketURL()
	bucketUrl, err := url.Parse(urlStr)
	if err != nil {
		log.Fatal(err)
	}
	// Create a new COS client
	cosClient = cos.NewClient(
		&cos.BaseURL{BucketURL: bucketUrl},
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  secretId,
				SecretKey: secretKey,
			},
		},
	)
}

func getTencentBucketURL() string {
	region := os.Getenv("COS_REGION")
	bucket := os.Getenv("COS_BUCKET")

	if region == "" || bucket == "" {
		log.Fatal("COS_REGION or COS_BUCKET is empty")
	}

	urlStr := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region)
	return urlStr
}

func getTencentCOSURL(filePath string) string {
	urlBaseStr := getTencentBucketURL()
	urlStr := fmt.Sprintf("%s/%s", urlBaseStr, filePath)
	return urlStr
}

func hasFileAtTencentCOS(filepath string) (bool, error) {
	return cosClient.Object.IsExist(context.Background(), filepath)
}

func uploadFileToTencentCOS(filePath string) error {
	log.Printf("[uploadFileToTencentCOS] Uploading fule %s...\n", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = cosClient.Object.Put(context.Background(), filePath, f, nil)
	if err != nil {
		return err
	}

	log.Println("[uploadFileToTencentCOS] upload file success " + filePath)
	return nil
}

var discordSession *discordgo.Session

func initDiscordBot() {
	discordToken := os.Getenv("DISCORD_TOKEN")
	// Create a new Discord session
	var err error
	discordSession, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	if channelID == "" {
		log.Fatal("Error DISCORD_CHANNEL_ID is empty")
	}
}

func sendMessageInDiscord(message string) {
	// Send a message to a Discord channel
	channelID := os.Getenv("DISCORD_CHANNEL_ID")

	_, err := discordSession.ChannelMessageSend(channelID, message)
	if err != nil {
		log.Fatalln("Error sending message:", err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
	initCOS()
	initDiscordBot()

	githubOwner := os.Getenv("GITHUB_OWNER")
	githubRepo := os.Getenv("GITHUB_REPO")
	githubToken := os.Getenv("GITHUB_TOKEN")

	if len(githubOwner) == 0 {
		log.Fatal("GITHUB_OWNER is not set")
	}
	if len(githubRepo) == 0 {
		log.Fatal("GITHUB_REPO is not set")
	}
	if len(githubToken) == 0 {
		log.Fatal("GITHUB_TOKEN is not set")
	}

	err = downloadLatestAssets(githubOwner, githubRepo, githubToken)
	if err != nil {
		log.Fatal(err)
	}
}
