package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	translator "github.com/zijiren233/google-translator"
)

type TranslationOutput struct {
	langCode string
	value    string
}

type OutPutMap struct {
	textKey      string
	translations map[string]string
}

func translate(text string, sourceLang string, destLang string, wg *sync.WaitGroup, ch chan<- TranslationOutput) {
	defer wg.Done()
	translated, err := translator.Translate(text, destLang, translator.TranslationParams{
		From: sourceLang,
	})
	if err != nil {
		fmt.Println(err.Error())
		ch <- TranslationOutput{destLang, ""}
		return
	}
	ch <- TranslationOutput{destLang, translated.Text}
}

func replaceSpacesWithUnderscore(text string) string {
	return strings.ReplaceAll(text, " ", "_")
}

func translateToMultipleLangs(text string, sourceLangCode string, SupportedLangs []string, wg *sync.WaitGroup, outputCh chan<- OutPutMap) {

	ch := make(chan TranslationOutput, len(SupportedLangs))
	fmt.Println("translating", text)
	defer wg.Done()
	wg1 := sync.WaitGroup{}
	for _, lang := range SupportedLangs {
		wg1.Add(1)
		go func() { translate(text, sourceLangCode, lang, &wg1, ch) }()
	}
	translations := make(map[string]string)
	for range SupportedLangs {
		translation := <-ch
		translations[translation.langCode] = translation.value
	}
	// fmt.Println(translations)
	wg1.Wait()
	outputCh <- OutPutMap{replaceSpacesWithUnderscore(text), translations}
}

// Read from input.json
func readInputFile() ([]string, string, []string, error) {
	inputFile, err := os.Open("input.json")
	if err != nil {
		return nil, "", nil, fmt.Errorf("Error opening input file: %s", err.Error())
	}
	defer inputFile.Close()

	// Decode JSON data
	var inputData struct {
		SourceLangCode    string   `json:"source_lang"`
		SourceTexts       []string `json:"source_texts"`
		SupporteLangCodes []string `json:"supported_langs"`
	}
	jsonDecoder := json.NewDecoder(inputFile)
	err = jsonDecoder.Decode(&inputData)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Error decoding JSON data: %s", err.Error())
	}

	// Assign source_texts array to sourceWords variable
	sourceWords := inputData.SourceTexts
	// Remove duplicates from langCodes
	langCodes := inputData.SupporteLangCodes
	sourceLangCode := inputData.SourceLangCode
	uniqueLangCodes := make([]string, 0, len(langCodes))
	seen := make(map[string]bool)
	for _, code := range langCodes {
		if !seen[code] {
			uniqueLangCodes = append(uniqueLangCodes, code)
			seen[code] = true
		}
	}
	langCodes = uniqueLangCodes
	return sourceWords, sourceLangCode, langCodes, nil
}

func main() {
	now := time.Now()
	wg := sync.WaitGroup{}
	sourceWords, sourceLangCode, langCodes, err := readInputFile()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("total lines", len(sourceWords))
	outputCh := make(chan OutPutMap, len(sourceWords))
	for _, word := range sourceWords {
		wg.Add(1)
		go func() { translateToMultipleLangs(word, sourceLangCode, langCodes, &wg, outputCh) }()
	}
	go func() {
		wg.Wait()
		close(outputCh)
	}()
	outputMap := make(map[string]map[string]string)
	for range sourceWords {
		result := <-outputCh
		outputMap[result.textKey] = result.translations
	}
	fmt.Println()
	fmt.Println("************************************************************")
	fmt.Println("* Translated", len(sourceWords), "lines to", len(langCodes), "languages")
	fmt.Println("* Total translations:", len(langCodes)*len(sourceWords))
	fmt.Println("* Time took:", time.Since(now))
	fmt.Println("************************************************************")
	fmt.Println()

	jsonData, err := json.Marshal(outputMap)
	if err != nil {
		fmt.Println("Error marshaling outputMap to JSON:", err.Error())
		return
	}

	outfile, err := os.Create("translations_output.json")
	if err != nil {
		fmt.Println("Error creating outfile:", err.Error())
		return
	}
	defer outfile.Close()

	_, err = outfile.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing JSON data to outfile:", err.Error())
		return
	}
	fmt.Println("translations written to translations_output.json")
}
