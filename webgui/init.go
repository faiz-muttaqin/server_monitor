package webgui

import (
	"bytes"
	"fmt"
	"html/template"
	"sync"

	"embed"

	"github.com/sirupsen/logrus"
)

//go:embed *.html
var embeddedTemplates embed.FS

var (
	tmplCache *template.Template
	mu        sync.Mutex // For thread safety
)

func Init() {
	var err error

	// Lock mutex to ensure thread safety
	mu.Lock()
	defer mu.Unlock()
	// List embedded files
	files, err := embeddedTemplates.ReadDir(".")
	if err != nil {
		logrus.Error(err)
		fmt.Println("Error reading embedded files:", err)
	} else {
		fmt.Println("Embedded template files:")
		for _, file := range files {
			fmt.Println(" -", file.Name())
		}
	}

	// Parse the templates during initialization
	tmplCache, err = template.ParseFS(embeddedTemplates, "*.html")
	if err != nil {
		logrus.Error(err)
		fmt.Println("Error parsing templates:", err)
	}
}

// RenderTemplateToString renders a template to a string
func RenderTemplateToString(templateName string, data interface{}) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	if tmplCache == nil {
		return "", fmt.Errorf("template cache is not initialized")
	}

	// Create a buffer to capture the template output
	var renderedTemplate bytes.Buffer

	// Execute the template
	err := tmplCache.ExecuteTemplate(&renderedTemplate, templateName, data)
	if err != nil {
		logrus.Error(err)
		return "", err
	}

	return renderedTemplate.String(), nil
}
