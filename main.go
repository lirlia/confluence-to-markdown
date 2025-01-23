package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	markdown "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

type ConfluencePage struct {
	Title string `json:"title"`
	Body  struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, os.ModePerm)
}

func sanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "_")
	}
	return name
}

func fetchConfluencePage(apiURL, email, apiToken, pageID string) (*ConfluencePage, error) {
	client := resty.New()
	resp, err := client.R().
		SetBasicAuth(email, apiToken).
		SetHeader("Content-Type", "application/json").
		SetQueryParams(map[string]string{
			"expand": "body.storage",
		}).
		Get(fmt.Sprintf("%s/content/%s", apiURL, pageID))

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page %s: %s", pageID, resp.Status())
	}

	var page ConfluencePage
	if err := json.Unmarshal(resp.Body(), &page); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &page, nil
}

func saveMarkdown(outputDir, title, content string) error {
	// HTML を Markdown に変換
	markdownContent, err := htmlToMarkdown(content)
	if err != nil {
		return err
	}

	filePath := filepath.Join(outputDir, sanitizeFilename(title)+".md")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(markdownContent)
	return err
}

func htmlToMarkdown(htmlContent string) (string, error) {
	// html-to-markdown を使って HTML を Markdown に変換
	converter := markdown.NewConverter("", true, nil)
	converter.Use(plugin.GitHubFlavored()) // GitHub風のMarkdownサポート
	converter.Use(plugin.Table())          // テーブルサポート

	markdownContent, err := converter.ConvertString(htmlContent)
	if err != nil {
		return "", err
	}
	return markdownContent, nil
}

func findImageURLs(content string) []string {
	var urls []string
	startTag := "<img src=\""
	endTag := "\""
	startIdx := 0

	for {
		startIdx = strings.Index(content[startIdx:], startTag)
		if startIdx == -1 {
			break
		}
		startIdx += len(startTag)
		endIdx := strings.Index(content[startIdx:], endTag)
		if endIdx == -1 {
			break
		}
		imgURL := content[startIdx : startIdx+endIdx]
		urls = append(urls, imgURL)
		startIdx += endIdx
	}

	return urls
}

func downloadImages(outputDir, content, apiURL, email, apiToken string) error {
	imageURLs := findImageURLs(content)

	for _, imgURL := range imageURLs {
		// 画像をダウンロード
		fileName := filepath.Base(imgURL)
		outputPath := filepath.Join(outputDir, fileName)

		client := resty.New()
		resp, err := client.R().
			SetBasicAuth(email, apiToken).
			SetDoNotParseResponse(true).
			Get(imgURL)

		if err != nil {
			fmt.Printf("Failed to download image %s: %v\n", imgURL, err)
			continue
		}

		outFile, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("Failed to save image %s: %v\n", imgURL, err)
			continue
		}

		_, err = io.Copy(outFile, resp.RawBody())
		if err != nil {
			fmt.Printf("Failed to write image %s: %v\n", imgURL, err)
		}

		outFile.Close()
		resp.RawBody().Close()
	}

	return nil
}

func main() {
	var (
		backupDir string
		apiURL    string
		email     string
		apiToken  string
		pageIDs   []string
	)

	// pflag を使用してコマンドライン引数を定義
	pflag.StringVar(&backupDir, "backup-dir", "./backup", "Directory to save backups")
	pflag.StringVar(&apiURL, "api-url", "", "Base URL of the Confluence API (required)")
	pflag.StringVar(&email, "email", "", "Email address for API authentication (required)")
	pflag.StringVar(&apiToken, "api-token", "", "API token for authentication (required)")
	pflag.StringSliceVar(&pageIDs, "page-ids", nil, "List of Confluence page IDs to fetch (required)")

	pflag.Parse()

	// 必須引数のバリデーション
	if apiURL == "" || email == "" || apiToken == "" || len(pageIDs) == 0 {
		fmt.Println("Missing required arguments. Use --help for usage.")
		os.Exit(1)
	}

	for idx, pageID := range pageIDs {
		fmt.Printf("Fetching page %d: %s\n", idx+1, pageID)

		page, err := fetchConfluencePage(apiURL, email, apiToken, pageID)
		if err != nil {
			fmt.Printf("Error fetching page %s: %v\n", pageID, err)
			continue
		}

		pageDir := filepath.Join(backupDir, fmt.Sprintf("%s(%s)", sanitizeFilename(page.Title), pageID))
		if err := ensureDir(pageDir); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", pageDir, err)
			continue
		}

		// Markdown を保存
		err = saveMarkdown(pageDir, page.Title, page.Body.Storage.Value)
		if err != nil {
			fmt.Printf("Error saving markdown for page %s: %v\n", pageID, err)
			continue
		}
		fmt.Printf("Saved markdown to %s/%s.md\n", pageDir, sanitizeFilename(page.Title))

		// 画像をダウンロード
		err = downloadImages(pageDir, page.Body.Storage.Value, apiURL, email, apiToken)
		if err != nil {
			fmt.Printf("Error downloading images for page %s: %v\n", pageID, err)
			continue
		}
	}

	fmt.Println("Completed!")
}
