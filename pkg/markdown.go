package pkg

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"sync"

	"time"

	"github.com/justmiles/go-confluence"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	e "github.com/Benbentwo/go-markdown2confluence/pkg/extension"
	"github.com/Benbentwo/utils/util"
)

const (
	// DefaultEndpoint provides an example endpoint for users
	DefaultEndpoint = "https://mydomain.atlassian.net/wiki"

	// Parallelism determines how many files to convert and upload at a time
	Parallelism = 5
)

type MarkdownConverter struct {
	Parent         string `json:"parent"`
	SourceMarkdown string `json:"source"`
	Title          string `json:"title"`
}

// Markdown2Confluence stores the settings for each run
type Markdown2Confluence struct {
	Space          string              `json:"space"`
	Title          string              `json:"title"`
	File           string              `json:"file"`
	Ancestor       string              `json:"ancestor"`
	Debug          bool                `json:"debug"`
	Since          int                 `json:"since"`
	Username       string              `json:"username"`
	Password       string              `json:"password"`
	Endpoint       string              `json:"endpoint"`
	Parent         string              `json:"parent"`
	SourceMarkdown []string            `json:"source"` // TODO deprecate
	Sources        []MarkdownConverter `json:"sources"`

	client         *confluence.Client
	LoadFromConfig *[]string `json:"parent_config"`
	DryRun         bool
	RunAllFiles    string
}

const DefaultConfigFile = `./.github/confluence.yml`

func (m *Markdown2Confluence) LoadConfig() error {
	inputs := *m.LoadFromConfig
	inputs = append(inputs, DefaultConfigFile)
	for i := len(inputs) - 1; i >= 0; i-- {
		fileName := inputs[i]
		m.LoadSingleConfig(fileName) // get parents
		parent_configs := *m.LoadFromConfig
		for i := len(parent_configs) - 1; i >= 0; i-- {
			parentFileName := inputs[i]
			m.LoadSingleConfig(parentFileName)
		}
		m.LoadSingleConfig(fileName)
	}

	return nil
}

func (m *Markdown2Confluence) LoadSingleConfig(fileName string) error {
	if fileName == "" {
		return fmt.Errorf("no file passed to load")
	}
	exists, err := util.FileExists(fileName)
	if err != nil {
		return fmt.Errorf("Could not check if file exists %s due to %s", fileName, err)
	}
	if exists {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return fmt.Errorf("Failed to load file %s due to %s", fileName, err)
		}
		err = yaml.Unmarshal(data, m)
		if err != nil {
			return fmt.Errorf("Failed to unmaxrshal YAML file %s due to %s", fileName, err)
		}
	}
	return nil
}

func (m *Markdown2Confluence) RunAllConfigs() map[string]error {
	var mapFileToError = make(map[string]error, 0)
	targetFile := m.RunAllFiles
	filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if info.Name() == targetFile {
				fmt.Println("adding: ", path)
				mapFileToError[path] = err
			}
			return nil
		})

	for file, _ := range mapFileToError {
		var loadConfig = make([]string, 0)
		loadConfig = append(loadConfig, file)
		individualConfig := &Markdown2Confluence{}
		if m.DryRun {
			individualConfig.DryRun = true
		}
		individualConfig.Endpoint = m.Endpoint
		individualConfig.Password = m.Password
		individualConfig.Username = m.Username
		individualConfig.Space = m.Space

		individualConfig.LoadFromConfig = &loadConfig

		fmt.Printf("Running config for %s\n", util.ColorStatus(file))
		err := individualConfig.DefaultRun()
		mapFileToError[file] = err
	}

	return mapFileToError
}

func (m *Markdown2Confluence) DefaultRun() error {
	err := m.LoadConfig()
	if err != nil {
		return err
	}
	m.SourceEnvironmentVariables()
	// Validate the arguments
	err = m.Validate()
	if err != nil {
		return err
	}

	if m.DryRun {
		m.PrintMe()
		return nil
	}
	errors := m.Run()
	for _, err := range errors {
		fmt.Println()
		fmt.Println(err)
	}
	if len(errors) > 0 {
		return errors[0]
	}
	return nil

}

func (m *Markdown2Confluence) PrintMe() {
	fmt.Printf("\tSpace          \t%s\n\tTitle          \t%s\n\tFile           \t%s\n\tAncestor       \t%s\n\tDebug          \t%s\n\tSince          \t%s\n\tUsername       \t%s\n\tPassword       \t%s\n\tEndpoint       \t%s\n\tParent         \t%s\n\tSourceMarkdown \t%s\n\n",
		util.ColorInfo(m.Space), util.ColorInfo(m.Title), util.ColorInfo(m.File), util.ColorInfo(m.Ancestor), util.ColorInfo(m.Debug), util.ColorInfo(m.Since), util.ColorInfo(m.Username), util.ColorInfo(m.Password), util.ColorInfo(m.Endpoint), util.ColorInfo(m.Parent), util.ColorInfo(m.SourceMarkdown))
}

// CreateClient returns a new markdown clietn
func (m *Markdown2Confluence) CreateClient() {
	m.client = new(confluence.Client)
	m.client.Username = m.Username
	m.client.Password = m.Password
	m.client.Endpoint = m.Endpoint
	m.client.Debug = m.Debug
}

// SourceEnvironmentVariables overrides Markdown2Confluence with any environment variables that are set
//  - CONFLUENCE_USERNAME
//  - CONFLUENCE_PASSWORD
//  - CONFLUENCE_ENDPOINT
func (m *Markdown2Confluence) SourceEnvironmentVariables() {
	var s string
	s = os.Getenv("CONFLUENCE_USERNAME")
	if s != "" {
		m.Username = s
	}

	s = os.Getenv("CONFLUENCE_PASSWORD")
	if s != "" {
		m.Password = s
	}

	s = os.Getenv("CONFLUENCE_ENDPOINT")
	if s != "" {
		m.Endpoint = s
	}
}

// Validate required configs are set
func (m Markdown2Confluence) Validate() error {
	if m.Space == "" {
		return fmt.Errorf("--space is not defined")
	}
	if m.Username == "" {
		return fmt.Errorf("--username is not defined")
	}
	if m.Password == "" {
		return fmt.Errorf("--password is not defined")
	}
	if m.Endpoint == "" {
		return fmt.Errorf("--endpoint is not defined")
	}
	if m.Endpoint == DefaultEndpoint {
		return fmt.Errorf("--endpoint is not defined")
	}
	if len(m.SourceMarkdown) == 0 {
		return fmt.Errorf("please pass a markdown file or directory of markdown files")
	}
	if len(m.SourceMarkdown) > 1 && m.Title != "" {
		return fmt.Errorf("You can not set the title for multiple files")
	}
	return nil
}

// Run the sync
func (m *Markdown2Confluence) Run() []error {
	var markdownFiles []MarkdownFile
	var now = time.Now()
	m.CreateClient()

	for _, f := range m.SourceMarkdown {
		file, err := os.Open(f)
		defer file.Close()
		if err != nil {
			return []error{fmt.Errorf("Error opening file %s", err)}
		}

		stat, err := file.Stat()
		if err != nil {
			return []error{fmt.Errorf("Error reading file meta %s", err)}
		}

		var md MarkdownFile

		if stat.IsDir() {

			// prevent someone from accidently uploading everything under the same title
			if m.Title != "" {
				return []error{fmt.Errorf("--title not supported for directories")}
			}

			err := filepath.Walk(f,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if strings.HasSuffix(path, ".md") {

						// Only include this file if it was modified m.Since minutes ago
						if m.Since != 0 {
							if info.ModTime().Unix() < now.Add(time.Duration(m.Since*-1)*time.Minute).Unix() {
								if m.Debug {
									fmt.Printf("skipping %s: last modified %s\n", info.Name(), info.ModTime())
								}
								return nil
							}
						}

						md := MarkdownFile{
							Path:    path,
							Parents: deleteFromSlice(strings.Split(filepath.Dir(strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(f))), "/"), "."),
							Title:   strings.TrimSuffix(filepath.Base(path), ".md"),
						}

						if m.Parent != "" {
							md.Parents = append([]string{m.Parent}, md.Parents...)
							md.Parents = deleteEmpty(md.Parents)
						}

						markdownFiles = append(markdownFiles, md)

					}
					return nil
				})
			if err != nil {
				return []error{fmt.Errorf("Unable to walk path: %s", f)}
			}

		} else {
			md = MarkdownFile{
				Path:  f,
				Title: m.Title,
			}

			if md.Title == "" {
				md.Title = strings.TrimSuffix(filepath.Base(f), ".md")
			}

			if m.Parent != "" {
				md.Parents = append([]string{m.Parent}, md.Parents...)
				md.Parents = deleteEmpty(md.Parents)
			}

			markdownFiles = append(markdownFiles, md)
		}
	}

	for _, source := range m.Sources {
		f := source.SourceMarkdown

		file, err := os.Open(f)
		defer file.Close()
		if err != nil {
			return []error{fmt.Errorf("Error opening file %s", err)}
		}

		stat, err := file.Stat()
		if err != nil {
			return []error{fmt.Errorf("Error reading file meta %s", err)}
		}

		var md MarkdownFile

		if stat.IsDir() {

			// prevent someone from accidently uploading everything under the same title
			if m.Title != "" {
				return []error{fmt.Errorf("--title not supported for directories")}
			}

			err := filepath.Walk(f,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if strings.HasSuffix(path, ".md") {

						// Only include this file if it was modified m.Since minutes ago
						if m.Since != 0 {
							if info.ModTime().Unix() < now.Add(time.Duration(m.Since*-1)*time.Minute).Unix() {
								if m.Debug {
									fmt.Printf("skipping %s: last modified %s\n", info.Name(), info.ModTime())
								}
								return nil
							}
						}

						md := MarkdownFile{
							Path:    path,
							Parents: deleteFromSlice(strings.Split(filepath.Dir(strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(f))), "/"), "."),
							Title:   strings.TrimSuffix(filepath.Base(path), ".md"),
						}

						if source.Parent != "" {
							md.Parents = append([]string{m.Parent}, md.Parents...)
							md.Parents = deleteEmpty(md.Parents)
						}

						markdownFiles = append(markdownFiles, md)

					}
					return nil
				})
			if err != nil {
				return []error{fmt.Errorf("Unable to walk path: %s", f)}
			}

		} else {
			md = MarkdownFile{
				Path:  f,
				Title: source.Title,
			}

			if md.Title == "" {
				md.Title = strings.TrimSuffix(filepath.Base(f), ".md")
			}

			if source.Parent != "" {
				md.Parents = append([]string{m.Parent, source.Parent}, md.Parents...)
				md.Parents = deleteEmpty(md.Parents)
			}

			markdownFiles = append(markdownFiles, md)
		}

	}
	var (
		wg    = sync.WaitGroup{}
		queue = make(chan MarkdownFile)
	)

	var errors []error

	// Process the queue
	for worker := 0; worker < Parallelism; worker++ {
		wg.Add(1)
		go m.queueProcessor(&wg, &queue, &errors)
	}

	for _, markdownFile := range markdownFiles {

		// Create parent pages synchronously
		if len(markdownFile.Parents) > 0 {
			var err error
			markdownFile.Ancestor, err = markdownFile.FindOrCreateAncestors(m)
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}

		queue <- markdownFile
	}

	close(queue)

	wg.Wait()

	return errors
}

func (m *Markdown2Confluence) queueProcessor(wg *sync.WaitGroup, queue *chan MarkdownFile, errors *[]error) {
	defer wg.Done()

	for markdownFile := range *queue {
		url, err := markdownFile.Upload(m)
		if err != nil {
			*errors = append(*errors, fmt.Errorf("Unable to upload markdown file, %s: \n\t%s", markdownFile.Path, err))
		}
		fmt.Printf("%s: %s\n", markdownFile.FormattedPath(), url)
	}
}

func validateInput(s string, msg string) {
	if s == "" {
		fmt.Println(msg)
		os.Exit(1)
	}
}

func renderContent(filePath, s string) (content string, images []string, err error) {
	confluenceExtension := e.NewConfluenceExtension(filePath)
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.DefinitionList),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(
			confluenceExtension,
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		return "", nil, err
	}

	return buf.String(), confluenceExtension.Images(), nil
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func deleteFromSlice(s []string, del string) []string {
	for i, v := range s {
		if v == del {
			s = append(s[:i], s[i+1:]...)
			break
		}
	}
	return s
}
