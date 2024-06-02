package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "os"
    "path"
    "strconv"
    "strings"
    "time"

    "github.com/spf13/cobra"
)

const (
    configFileName         = "slack.config.json"
    emojiFileName          = "slack.emoji.json"
    slackUploadURL         = "https://slack.com/api/files.getUploadURLExternal"
    slackCompleteUploadURL = "https://slack.com/api/files.completeUploadExternal"
)

var buildTime string

type Config struct {
    SlackBotToken    string            `json:"slack_bot_token"`
    SlackUserToken   string            `json:"slack_user_token"`
    ChannelID        string            `json:"channel_id"`
    ServerBotUserID  string            `json:"server_bot_user_id"`
    UserCache        map[string]string `json:"user_cache"`
    DefaultShowLimit int               `json:"default_show_limit"`
    DefaultEmoji     string            `json:"default_emoji"`
}

var config Config
var emojiList map[string]string

func loadConfig() error {
    configFile, err := os.Open(configFileName)
    if err != nil {
        return fmt.Errorf("could not open config file: %v", err)
    }
    defer configFile.Close()

    err = json.NewDecoder(configFile).Decode(&config)
    if err != nil {
        return fmt.Errorf("could not decode config JSON: %v", err)
    }

    return nil
}

func loadEmojiConfig() error {
    emojiFile, err := os.Open(emojiFileName)
    if err != nil {
        return fmt.Errorf("could not open emoji file: %v", err)
    }
    defer emojiFile.Close()

    err = json.NewDecoder(emojiFile).Decode(&emojiList)
    if err != nil {
        return fmt.Errorf("could not decode emoji JSON: %v", err)
    }

    return nil
}

func saveConfig() error {
    configFile, err := os.Create(configFileName)
    if err != nil {
        return fmt.Errorf("could not create config file: %v", err)
    }
    defer configFile.Close()

    configBytes, err := json.MarshalIndent(config, "", "    ")
    if err != nil {
        return fmt.Errorf("could not marshal config JSON: %v", err)
    }

    _, err = configFile.Write(configBytes)
    if err != nil {
        return fmt.Errorf("could not write to config file: %v", err)
    }

    return nil
}

func createConfig() error {
    config = Config{
        ChannelID:       "your_channel_id",
        SlackBotToken:   "your_slack_bot_token",
        SlackUserToken:  "your_slack_user_token",
        ServerBotUserID: "your_server_bot_user_id",
        UserCache: map[string]string{
            "U075JAXRYV7": "Bot",
        },
        DefaultShowLimit: 20,
        DefaultEmoji:     "white-check-mark",
    }

    return saveConfig()
}

func checkAndLoadConfig() {
    if _, err := os.Stat(configFileName); os.IsNotExist(err) {
        fmt.Println("Config file not found, creating a new one with template.")
        err := createConfig()
        if err != nil {
            fmt.Println("Error creating config file:", err)
            os.Exit(1)
        }
        fmt.Printf("Please edit %s with your configuration.\n", configFileName)
        os.Exit(0)
    } else {
        err := loadConfig()
        if err != nil {
            fmt.Println("Error loading config file:", err)
            os.Exit(1)
        }
    }

    err := loadEmojiConfig()
    if err != nil {
        fmt.Println("Error loading emoji config file:", err)
        os.Exit(1)
    }
}

type SlackMessage struct {
    Text string `json:"text"`
}

type SlackReaction struct {
    Name  string   `json:"name"`
    Count int      `json:"count"`
    Users []string `json:"users"`
}

type SlackMessageReply struct {
    UserID    string          `json:"user"`
    UserName  string          `json:"user_name"`
    Text      string          `json:"text"`
    Ts        string          `json:"ts"`
    Reactions []SlackReaction `json:"reactions,omitempty"`
    Files     []struct {
        URLPrivate string `json:"url_private"`
        Name       string `json:"name"`
        Mimetype   string `json:"mimetype"`
    } `json:"files,omitempty"`
    Edited struct {
        User string `json:"user"`
    } `json:"edited,omitempty"`
}

type SlackMessageItem struct {
    UserID    string              `json:"user"`
    UserName  string              `json:"user_name"`
    Text      string              `json:"text"`
    Ts        string              `json:"ts"`
    ThreadTS  string              `json:"thread_ts,omitempty"`
    Replies   []SlackMessageReply `json:"replies,omitempty"`
    Reactions []SlackReaction     `json:"reactions,omitempty"`
    Files     []struct {
        URLPrivate string `json:"url_private"`
        Name       string `json:"name"`
        Mimetype   string `json:"mimetype"`
    } `json:"files,omitempty"`
    Edited struct {
        User string `json:"user"`
    } `json:"edited,omitempty"`
}

type SlackMessagesResponse struct {
    Messages []SlackMessageItem `json:"messages"`
    HasMore  bool               `json:"has_more"`
    ResponseMetadata struct {
        NextCursor string `json:"next_cursor"`
    } `json:"response_metadata"`
}

type GetUploadURLResponse struct {
    OK        bool   `json:"ok"`
    UploadURL string `json:"upload_url"`
    FileID    string `json:"file_id"`
}

type UserProfile struct {
    OK   bool `json:"ok"`
    User struct {
        RealName string `json:"real_name"`
    } `json:"user"`
}

func getUserName(userID string, userCache map[string]string) string {
    if userID == "" {
        return "Unknown"
    }

    if name, exists := userCache[userID]; exists {
        return name
    }
    if name, exists := config.UserCache[userID]; exists {
        userCache[userID] = name 
        return name
    }

    apiURL := fmt.Sprintf("https://slack.com/api/users.info?user=%s", userID)
    req, _ := http.NewRequest("GET", apiURL, nil)
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error fetching user profile:", err)
        return "Unknown"
    }
    defer resp.Body.Close()

    var userProfile UserProfile
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Error reading user profile response:", err)
        return "Unknown"
    }

    err = json.Unmarshal(body, &userProfile)
    if err != nil {
        fmt.Println("Error decoding user profile JSON:", err)
        return "Unknown"
    }

    if userProfile.OK {
        name := userProfile.User.RealName
        userCache[userID] = name
        config.UserCache[userID] = name
        saveConfig()
        return name
    }

    return "Unknown"
}

func sendMessage(message, threadTS string) error {
    slackMessage := map[string]string{
        "channel": config.ChannelID,
        "text":    message,
    }

    if threadTS != "" {
        slackMessage["thread_ts"] = threadTS
    }

    payload, _ := json.Marshal(slackMessage)

    req, _ := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(payload))
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error sending message:", err)
        return err
    }
    defer resp.Body.Close()

    fmt.Println("Message sent successfully")
    return nil
}

func fetchMessages(limit int, dateRange, search, filter string, showFilesOnly bool) {
    var cursor string
    var hasMore bool

    if dateRange == "" {
        hasMore = false
    } else {
        hasMore = true
    }

    var oldest, latest string
    if dateRange != "" {
        var oldestInt, latestInt int64
        dates := strings.Split(dateRange, ":")
        if len(dates) == 1 {
            parsedDate, err := time.Parse("2006-01-02", dates[0])
            if err != nil {
                fmt.Println("Invalid date format. Use YYYY-MM-DD.")
                return
            }
            oldestInt = parsedDate.Unix()
            latestInt = parsedDate.AddDate(0, 0, 1).Unix() - 1
        } else if len(dates) == 2 {
            parsedStartDate, err := time.Parse("2006-01-02", dates[0])
            if err != nil {
                fmt.Println("Invalid start date format. Use YYYY-MM-DD.")
                return
            }
            parsedEndDate, err := time.Parse("2006-01-02", dates[1])
            if err != nil {
                fmt.Println("Invalid end date format. Use YYYY-MM-DD.")
                return
            }
            oldestInt = parsedStartDate.Unix()
            latestInt = parsedEndDate.AddDate(0, 0, 1).Unix() - 1
        } else {
            fmt.Println("Invalid date range format. Use YYYY-MM-DD or YYYY-MM-DD:YYYY-MM-DD.")
            return
        }
        oldest = strconv.FormatInt(oldestInt, 10)
        latest = strconv.FormatInt(latestInt, 10)
    }

    userCache := make(map[string]string)
    for k, v := range config.UserCache {
        userCache[k] = v
    }

    var messages []string
    indent := strings.Repeat(" ", 40)
    redColorStart := "\033[91m"
    resetColor := "\033[0m"
    defaultColorStart := "\033[39m"

    totalFetched := 0

    for {
        apiURL := fmt.Sprintf("https://slack.com/api/conversations.history?channel=%s&limit=%d", config.ChannelID, limit)
        if oldest != "" && latest != "" {
            apiURL = fmt.Sprintf("%s&oldest=%s&latest=%s", apiURL, oldest, latest)
        }
        if cursor != "" {
            apiURL = fmt.Sprintf("%s&cursor=%s", apiURL, cursor)
        }

        req, _ := http.NewRequest("GET", apiURL, nil)
        req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            fmt.Println("Error getting messages:", err)
            return
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
            fmt.Println("Error reading response body:", err)
            return
        }

        var messagesResponse SlackMessagesResponse
        err = json.Unmarshal(body, &messagesResponse)
        if err != nil {
            fmt.Println("Error unmarshaling response:", err)
            return
        }

        for i, j := 0, len(messagesResponse.Messages)-1; i < j; i, j = i+1, j-1 {
            messagesResponse.Messages[i], messagesResponse.Messages[j] = messagesResponse.Messages[j], messagesResponse.Messages[i]
        }

        for _, msg := range messagesResponse.Messages {
            if totalFetched >= limit {
                hasMore = false
                break
            }
            if filter == "" || strings.Contains(msg.Text, filter) {
                if showFilesOnly && len(msg.Files) == 0 {
                    continue
                }
                userName := getUserName(msg.UserID, userCache)
                edited := ""
                if msg.Edited.User != "" {
                    edited = " (edited)"
                }
                reactions := getReactionsString(msg.Reactions)
                textLines := strings.Split(msg.Text, "\n")
                for i, line := range textLines {
                    if search != "" {
                        line = strings.ReplaceAll(line, search, fmt.Sprintf("%s%s%s%s", redColorStart, search, resetColor, defaultColorStart))
                    } else if strings.Contains(msg.Text, filter) {
                        line = strings.ReplaceAll(line, filter, fmt.Sprintf("%s%s%s%s", redColorStart, filter, resetColor, defaultColorStart))
                    }
                    if i == 0 {
                        messages = append(messages, fmt.Sprintf("%s (%s) %s: %s%s%s%s\n", msg.Ts, formatTimestamp(msg.Ts), userName, defaultColorStart, line, edited, reactions))
                    } else {
                        messages = append(messages, fmt.Sprintf("%s%s%s\n", indent, defaultColorStart, line))
                    }
                }
                for _, file := range msg.Files {
                    fileNameColor := "\033[94m"
                    if strings.HasPrefix(file.Mimetype, "image/") {
                        fileNameColor = "\033[91m"
                    }
                    fileEntry := fmt.Sprintf("  - File: %s%s%s (\033[36m%s\033[0m)\n", fileNameColor, file.Name, resetColor, file.URLPrivate)
                    if search != "" {
                        fileEntry = fmt.Sprintf("  - File: %s (%s)\n", file.Name, file.URLPrivate)
                    }
                    messages = append(messages, fileEntry)
                }
                if msg.ThreadTS != "" {
                    replies, err := getThreadReplies(msg.ThreadTS, filter, search, userCache)
                    if err == nil {
                        for _, reply := range replies {
                            replyUserName := getUserName(reply.UserID, userCache)
                            edited = ""
                            if reply.Edited.User != "" {
                                edited = " (edited)"
                            }
                            reactions = getReactionsString(reply.Reactions)
                            textLines = strings.Split(reply.Text, "\n")
                            for i, line := range textLines {
                                if search != "" {
                                    line = strings.ReplaceAll(line, search, fmt.Sprintf("%s%s%s%s", redColorStart, search, resetColor, defaultColorStart))
                                }
                                if i == 0 {
                                    messages = append(messages, fmt.Sprintf("  ↳ %s (%s) %s: %s%s%s%s\n", reply.Ts, formatTimestamp(reply.Ts), replyUserName, defaultColorStart, line, edited, reactions))
                                } else {
                                    messages = append(messages, fmt.Sprintf("%s%s%s\n", indent, defaultColorStart, line))
                                }
                            }
                            for _, file := range reply.Files {
                                fileNameColor := "\033[94m"
                                if strings.HasPrefix(file.Mimetype, "image/") {
                                    fileNameColor = "\033[91m"
                                }
                                fileEntry := fmt.Sprintf("    - File: %s%s%s (\033[36m%s\033[0m)\n", fileNameColor, file.Name, resetColor, file.URLPrivate)
                                if search != "" {
                                    fileEntry = fmt.Sprintf("    - File: %s (%s)\n", file.Name, file.URLPrivate)
                                }
                                messages = append(messages, fileEntry)
                            }
                        }
                    }
                }
                totalFetched++
            }
        }

        hasMore = messagesResponse.HasMore && cursor != "" && dateRange != ""
        cursor = messagesResponse.ResponseMetadata.NextCursor

        if !hasMore {
            break
        }
    }

    for i := 0; i < len(messages); i++ {
        fmt.Print(messages[i])
    }
}
func formatTimestamp(ts string) string {
    tsFloat, _ := strconv.ParseFloat(ts, 64)
    timeStamp := time.Unix(int64(tsFloat), 0)
    return timeStamp.Format("2006-01-02 15:04:05")
}

func getThreadReplies(threadTs, filter, search string, userCache map[string]string) ([]SlackMessageReply, error) {
    apiURL := fmt.Sprintf("https://slack.com/api/conversations.replies?channel=%s&ts=%s", config.ChannelID, threadTs)
    req, _ := http.NewRequest("GET", apiURL, nil)
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var threadResponse struct {
        Messages []SlackMessageItem `json:"messages"`
    }
    json.NewDecoder(resp.Body).Decode(&threadResponse)

    var replies []SlackMessageReply
    for _, msg := range threadResponse.Messages {
        if msg.Ts != threadTs {
            if filter == "" || strings.Contains(msg.Text, filter) {
                replies = append(replies, SlackMessageReply{
                    UserID:    msg.UserID,
                    UserName:  getUserName(msg.UserID, userCache),
                    Text:      msg.Text,
                    Ts:        msg.Ts,
                    Reactions: msg.Reactions,
                    Files:     msg.Files,
                })
            }
        }
    }

    return replies, nil
}

func getReactionsString(reactions []SlackReaction) string {
    var reactionsStr string
    for _, reaction := range reactions {
        emoji, exists := emojiList[reaction.Name]
        if (!exists) {
            emoji = reaction.Name
        }
        emojiUnicode, _ := unicodeToEmoji(emoji)
        reactionsStr += fmt.Sprintf(" %s:%d", emojiUnicode, reaction.Count)
    }
    return reactionsStr
}

func uploadFile(filePath string) {
    file, err := os.Open(filePath)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    fileInfo, _ := file.Stat()
    fileSize := fileInfo.Size()
    fileSizeStr := strconv.FormatInt(fileSize, 10)

    var b bytes.Buffer
    writer := multipart.NewWriter(&b)
    writer.WriteField("filename", fileInfo.Name())
    writer.WriteField("length", fileSizeStr)
    writer.WriteField("channels", config.ChannelID)
    writer.WriteField("token", config.SlackBotToken)
    writer.Close()

    req, _ := http.NewRequest("POST", slackUploadURL, &b)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    resp, _ := client.Do(req)

    body, _ := io.ReadAll(resp.Body)

    var uploadURLResponse GetUploadURLResponse
    json.NewDecoder(bytes.NewReader(body)).Decode(&uploadURLResponse)

    uploadURL := uploadURLResponse.UploadURL
    fileID := uploadURLResponse.FileID

    var fileBuffer bytes.Buffer
    fileWriter := multipart.NewWriter(&fileBuffer)
    filePart, _ := fileWriter.CreateFormFile("file", fileInfo.Name())
    io.Copy(filePart, file)
    fileWriter.Close()

    uploadReq, _ := http.NewRequest("POST", uploadURL, &fileBuffer)
    uploadReq.Header.Set("Content-Type", fileWriter.FormDataContentType())

    client.Do(uploadReq)

    completeUploadPayload := map[string]interface{}{
        "files": []map[string]string{
            {
                "id": fileID,
            },
        },
        "channel_id": config.ChannelID,
    }

    completeUploadBytes, _ := json.Marshal(completeUploadPayload)

    completeUploadReq, _ := http.NewRequest("POST", slackCompleteUploadURL, bytes.NewBuffer(completeUploadBytes))
    completeUploadReq.Header.Set("Content-Type", "application/json")
    completeUploadReq.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    completeUploadResp, err := client.Do(completeUploadReq)
    if err != nil {
        fmt.Println("Failed to complete file upload")
        return
    }
    defer completeUploadResp.Body.Close()

    finalBody, err := io.ReadAll(completeUploadResp.Body)
    if err != nil {
        fmt.Println("Failed to read complete upload response")
        return
    }

    var finalResponse map[string]interface{}
    err = json.Unmarshal(finalBody, &finalResponse)
    if err != nil {
        fmt.Println("Failed to decode complete upload response")
        return
    }

    files, ok := finalResponse["files"].([]interface{})
    if !ok || len(files) == 0 {
        fmt.Println("File information not found in response")
        return
    }

    fileInfoMap, ok := files[0].(map[string]interface{})
    if !ok {
        fmt.Println("File information not found in response")
        return
    }

    _, fileURLExists := fileInfoMap["permalink"].(string)
    _, fileImageURLExists := fileInfoMap["url_private"].(string)
    if !fileURLExists || !fileImageURLExists {
        fmt.Println("File URLs not found in response")
        return
    }

    time.Sleep(2 * time.Second)

    fmt.Println("File uploaded and shared to Slack")
}

func getFile(fileURL string) {
    req, _ := http.NewRequest("GET", fileURL, nil)
    req.Header.Set("Authorization", "Bearer "+config.SlackUserToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error getting file:", err)
        return
    }
    defer resp.Body.Close()

    parsedURL, err := url.Parse(fileURL)
    if err != nil {
        fmt.Println("Error parsing URL:", err)
        return
    }

    filename := path.Base(parsedURL.Path)

    if _, err := os.Stat(filename); err == nil {
        ext := path.Ext(filename)
        name := filename[:len(filename)-len(ext)]
        for i := 1; ; i++ {
            newName := fmt.Sprintf("%s(%d)%s", name, i, ext)
            if _, err := os.Stat(newName); os.IsNotExist(err) {
                filename = newName
                break
            }
        }
    }

    outFile, err := os.Create(filename)
    if err != nil {
        fmt.Println("Error creating file:", err)
        return
    }
    defer outFile.Close()

    io.Copy(outFile, resp.Body)
    fmt.Println("File downloaded successfully:", filename)
}

func addReaction(ts string, emoji string) error {
    if ts == "" {
        return fmt.Errorf("Error: ts (timestamp) is required")
    }

    if emoji == "" {
        if config.DefaultEmoji != "" {
            emoji = config.DefaultEmoji
        } else {
            emoji = "white_check_mark"
        }
    }

    payload := map[string]string{
        "channel":   config.ChannelID,
        "name":      emoji,
        "timestamp": ts,
    }

    payloadBytes, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", "https://slack.com/api/reactions.add", bytes.NewBuffer(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("Error adding reaction: %v", err)
    }
    defer resp.Body.Close()

    if err := handleError(resp.Body); err != nil {
        return fmt.Errorf("Error response from API: %v", err)
    }

    fmt.Println("Reaction added successfully")
    return nil
}

func removeReaction(ts string, emoji string) error {
    if ts == "" {
        return fmt.Errorf("Error: ts (timestamp) is required")
    }

    if emoji == "" {
        emoji = "white_check_mark"
    }

    payload := map[string]string{
        "channel":   config.ChannelID,
        "name":      emoji,
        "timestamp": ts,
    }

    payloadBytes, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", "https://slack.com/api/reactions.remove", bytes.NewBuffer(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("Error removing reaction: %v", err)
    }
    defer resp.Body.Close()

    if err := handleError(resp.Body); err != nil {
        return fmt.Errorf("Error response from API: %v", err)
    }

    fmt.Println("Reaction removed successfully")
    return nil
}

func updateMessage(ts, message string) error {
    payload := map[string]string{
        "channel": config.ChannelID,
        "ts":      ts,
        "text":    message,
    }

    payloadBytes, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", "https://slack.com/api/chat.update", bytes.NewBuffer(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error updating message:", err)
        return err
    }
    defer resp.Body.Close()

    fmt.Println("Message updated successfully")
    return nil
}

func deleteMessage(ts string) error {
    payload := map[string]string{
        "channel": config.ChannelID,
        "ts":      ts,
    }

    payloadBytes, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", "https://slack.com/api/chat.delete", bytes.NewBuffer(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+config.SlackBotToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error deleting message:", err)
        return err
    }
    defer resp.Body.Close()

    fmt.Println("Message deleted successfully")
    return nil
}

func handleError(body io.Reader) error {
    var response struct {
        OK    bool   `json:"ok"`
        Error string `json:"error"`
    }
    if err := json.NewDecoder(body).Decode(&response); err != nil {
        return err
    }
    if !response.OK {
        return fmt.Errorf(response.Error)
    }
    return nil
}

func handleEmoji(ts, emoji, add, del string) error {
    if add != "" {
        return addReaction(ts, add)
    } else if del != "" {
        return removeReaction(ts, del)
    } else {
        if emoji == "" {
            return listEmoji()
        } else {
            return addReaction(ts, emoji)
        }
    }
}

func listEmoji() error {
    if len(emojiList) == 0 {
        fmt.Println("Loading emoji list from file...")

        err := loadEmojiConfig()
        if err != nil {
            return fmt.Errorf("Error loading emoji list from file: %v", err)
        }

        fmt.Println("Emoji list loaded successfully")
    }

    fmt.Println("Emoji List:")
    for name, code := range emojiList {
        emoji, err := unicodeToEmoji(code)
        if err != nil {
            fmt.Printf("%s: %s (failed to convert)\n", name, code)
        } else {
            fmt.Printf("%s: %s\n", name, emoji)
        }
    }

    return nil
}

func unicodeToEmoji(unicodeStr string) (string, error) {
    unicodeStr = strings.ReplaceAll(unicodeStr, "&#x", "")
    unicodeStr = strings.ReplaceAll(unicodeStr, ";", "")
    runeValue, err := strconv.ParseInt(unicodeStr, 16, 32)
    if err != nil {
        return "", err
    }
    return string(rune(runeValue)), nil
}

func main() {
    checkAndLoadConfig()

    fmt.Printf("Slack CLI (build time: %s)\n", buildTime) // 빌드 시간 출력

    var rootCmd = &cobra.Command{Use: "slack"}

    var sendCmd = &cobra.Command{
        Use:   "send [message]",
        Short: "Send a message to Slack",
        Run: func(cmd *cobra.Command, args []string) {
            if len(args) < 1 {
                fmt.Println("Error: message is required")
                return
            }
            threadTS, _ := cmd.Flags().GetString("ts")
            sendMessage(args[0], threadTS)
        },
    }
    sendCmd.Flags().String("ts", "", "Thread timestamp")

    var showCmd = &cobra.Command{
        Use:   "show [limit]",
        Short: "Get messages from Slack",
        Args:  cobra.MaximumNArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            var limit int
            if len(args) > 0 {
                parsedLimit, err := strconv.Atoi(args[0])
                if err != nil {
                    fmt.Println("Invalid limit value, using default value.")
                    limit = config.DefaultShowLimit
                } else {
                    limit = parsedLimit
                }
            } else {
                limit, _ = cmd.Flags().GetInt("limit")
            }
            date, _ := cmd.Flags().GetString("date")
            search, _ := cmd.Flags().GetString("search")
            filter, _ := cmd.Flags().GetString("filter")
            showFilesOnly, _ := cmd.Flags().GetBool("files")
            fetchMessages(limit, date, search, filter, showFilesOnly)
        },
    }
    showCmd.Flags().String("date", "", "Date or date range for filtering messages (YYYY-MM-DD or YYYY-MM-DD:YYYY-MM-DD)")
    showCmd.Flags().String("search", "", "Keyword to search in messages")
    showCmd.Flags().String("filter", "", "Keyword to filter messages")
    showCmd.Flags().Int("limit", config.DefaultShowLimit, "Limit the number of messages to retrieve")
    showCmd.Flags().Bool("files", false, "Show only messages with files")

    var uploadCmd = &cobra.Command{
        Use:   "upload [filePath]",
        Short: "Upload a file to Slack",
        Run: func(cmd *cobra.Command, args []string) {
            if len(args) < 1 {
                fmt.Println("Error: file path is required")
                return
            }
            uploadFile(args[0])
        },
    }

    var downloadCmd = &cobra.Command{
        Use:   "download [fileURL]",
        Short: "Download a file from Slack",
        Run: func(cmd *cobra.Command, args []string) {
            if len(args) < 1 {
                fmt.Println("Error: file URL is required")
                return
            }
            getFile(args[0])
        },
    }

    var emojiCmd = &cobra.Command{
        Use:   "emoji [ts] [emoji]",
        Short: "Add or remove a reaction to a Slack message",
        Run: func(cmd *cobra.Command, args []string) {
            ts := ""
            emoji := ""
            if len(args) > 0 {
                ts = args[0]
            }
            if len(args) > 1 {
                emoji = args[1]
            }
            add, _ := cmd.Flags().GetString("add")
            del, _ := cmd.Flags().GetString("del")
            handleEmoji(ts, emoji, add, del)
        },
    }
    emojiCmd.Flags().String("add", "", "Add a reaction to the message")
    emojiCmd.Flags().String("del", "", "Remove a reaction from the message")

    var editCmd = &cobra.Command{
        Use:   "edit [ts] [message]",
        Short: "Update a Slack message",
        Run: func(cmd *cobra.Command, args []string) {
            if len(args) < 2 {
                fmt.Println("Error: ts (timestamp) and message are required")
                return
            }
            updateMessage(args[0], args[1])
        },
    }

    var deleteCmd = &cobra.Command{
        Use:   "delete [ts]",
        Short: "Delete a Slack message",
        Run: func(cmd *cobra.Command, args []string) {
            if len(args) < 1 {
                fmt.Println("Error: ts (timestamp) is required")
                return
            }
            deleteMessage(args[0])
        },
    }

    var examplesCmd = &cobra.Command{
        Use:   "examples",
        Short: "Show examples for all commands",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println(`Examples:
   ./slack show
   ./slack show 100
   ./slack show --date 2023-12-31
   ./slack show --date 2023-12-29:2023-12-31
   ./slack show --search keyword
   ./slack show 500 --search keyword
   ./slack show --filter keyword
   ./slack show 500 --filter keyword
   ./slack show --files
   ./slack send "Hello, Slack!"
   ./slack send "Hello, Slack!" --ts 1234567890.123456 (reply)
   ./slack edit --ts 1234567890.123456 --msg "Updated message"
   ./slack edit 1234567890.123456 "Updated message"
   ./slack delete --ts 1234567890.123456
   ./slack delete 1234567890.123456
   ./slack upload path/to/your/file.txt
   ./slack download https://file.url
   ./slack emoji
   ./slack emoji 1234567890.123456
   ./slack emoji 1234567890.123456 white-check-mark
   ./slack emoji 1234567890.123456 --add thumbsup
   ./slack emoji 1234567890.123456 --del white-check-mark`)
        },
    }
    
    rootCmd.AddCommand(sendCmd)
    rootCmd.AddCommand(showCmd)
    rootCmd.AddCommand(uploadCmd)
    rootCmd.AddCommand(downloadCmd)
    rootCmd.AddCommand(emojiCmd)
    rootCmd.AddCommand(editCmd)
    rootCmd.AddCommand(deleteCmd)
    rootCmd.AddCommand(examplesCmd)

    rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}