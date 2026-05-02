package catalog

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"brights/api/internal/domain"
)

func FindDefaultDataFile(candidates []string) (string, bool) {
	seen := make(map[string]struct{})
	patterns := append([]string{}, candidates...)
	patterns = append(patterns,
		"brights*.csv",
		filepath.Join("..", "brights*.csv"),
		"brights*.xlsx",
		filepath.Join("..", "brights*.xlsx"),
	)

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			abs, err := filepath.Abs(match)
			if err != nil {
				return match, true
			}
			return abs, true
		}
	}

	return "", false
}

func LoadWordsFromFile(path, subjectKey string) ([]domain.Word, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	header := make([]byte, 4)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	header = header[:n]

	if bytes.HasPrefix(header, []byte("PK")) {
		return loadWordsFromXLSX(path, subjectKey)
	}
	return loadWordsFromCSV(path, subjectKey)
}

func loadWordsFromCSV(path, subjectKey string) ([]domain.Word, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	for i := range headers {
		headers[i] = normalizeHeader(headers[i])
	}

	var words []domain.Word
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		row := make(map[string]string, len(headers))
		for idx, header := range headers {
			if idx < len(record) {
				row[header] = strings.TrimSpace(record[idx])
			}
		}
		word, ok := buildWord(row, subjectKey)
		if ok {
			words = append(words, word)
		}
	}
	return words, nil
}

func loadWordsFromXLSX(path, subjectKey string) ([]domain.Word, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	sharedStrings, err := readSharedStrings(reader.File)
	if err != nil {
		return nil, err
	}

	sheetFile := firstWorksheet(reader.File)
	if sheetFile == nil {
		return nil, errors.New("worksheet not found")
	}

	rc, err := sheetFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	headers := make(map[string]string)
	var words []domain.Word
	rowNumber := 0

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "row" {
			continue
		}

		var row xlsxRow
		if err := decoder.DecodeElement(&row, &start); err != nil {
			return nil, err
		}
		rowNumber++
		if rowNumber == 1 {
			for _, cell := range row.Cells {
				column := columnFromCellRef(cell.Reference)
				headers[column] = normalizeHeader(cell.Resolve(sharedStrings))
			}
			continue
		}

		data := make(map[string]string, len(headers))
		for _, cell := range row.Cells {
			column := columnFromCellRef(cell.Reference)
			header := headers[column]
			if header == "" {
				continue
			}
			data[header] = strings.TrimSpace(cell.Resolve(sharedStrings))
		}

		word, ok := buildWord(data, subjectKey)
		if ok {
			words = append(words, word)
		}
	}

	return words, nil
}

func buildWord(row map[string]string, subjectKey string) (domain.Word, bool) {
	term := strings.TrimSpace(row["en"])
	translation := strings.TrimSpace(row["zh"])
	if term == "" {
		return domain.Word{}, false
	}

	id, _ := strconv.ParseInt(strings.TrimSpace(row["id"]), 10, 64)
	classification, source := normalizeClassification(row["classification"])

	return domain.Word{
		LegacyID:       id,
		SubjectKey:     subjectKey,
		Term:           term,
		Translation:    translation,
		Classification: classification,
		Source:         source,
		Phonetics:      strings.TrimSpace(row["phonetics"]),
		Explanation:    strings.TrimSpace(row["explain"]),
	}, true
}

func normalizeClassification(raw string) (classification string, source string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Unclassified", ""
	}

	if looksLikeSourceLabel(raw) {
		return "Imported Collections", raw
	}

	return raw, ""
}

func looksLikeSourceLabel(value string) bool {
	if utf8.RuneCountInString(value) > 12 {
		return true
	}

	tokens := []string{
		"TOEFL", "IELTS", "BBC", "COCA", "MBA", "SAT", "GRE", "CET", "TEM", "KET", "PET",
		"VOCAB", "WORDS", "LEXICON", "PHRASE", "PHRASES",
	}

	upperValue := strings.ToUpper(value)
	for _, token := range tokens {
		if strings.Contains(upperValue, strings.ToUpper(token)) {
			return true
		}
	}

	for _, r := range value {
		if unicode.IsDigit(r) {
			return true
		}
	}

	return false
}

func normalizeHeader(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "\ufeff")
	return strings.ToLower(value)
}

func firstWorksheet(files []*zip.File) *zip.File {
	worksheetNames := make([]string, 0, len(files))
	lookup := make(map[string]*zip.File, len(files))
	for _, file := range files {
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") {
			worksheetNames = append(worksheetNames, file.Name)
			lookup[file.Name] = file
		}
	}
	sort.Strings(worksheetNames)
	if len(worksheetNames) == 0 {
		return nil
	}
	return lookup[worksheetNames[0]]
}

func readSharedStrings(files []*zip.File) ([]string, error) {
	var sharedFile *zip.File
	for _, file := range files {
		if file.Name == "xl/sharedStrings.xml" {
			sharedFile = file
			break
		}
	}
	if sharedFile == nil {
		return nil, nil
	}

	rc, err := sharedFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	values := make([]string, 0, 1024)

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "si" {
			continue
		}

		var item sharedStringItem
		if err := decoder.DecodeElement(&item, &start); err != nil {
			return nil, err
		}
		values = append(values, item.String())
	}

	return values, nil
}

func columnFromCellRef(ref string) string {
	var builder strings.Builder
	for _, r := range ref {
		if unicode.IsLetter(r) {
			builder.WriteRune(r)
			continue
		}
		break
	}
	return builder.String()
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Reference    string            `xml:"r,attr"`
	Type         string            `xml:"t,attr"`
	Value        string            `xml:"v"`
	InlineString *sharedStringItem `xml:"is"`
}

func (c xlsxCell) Resolve(sharedStrings []string) string {
	switch c.Type {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(c.Value))
		if err != nil || index < 0 || index >= len(sharedStrings) {
			return ""
		}
		return sharedStrings[index]
	case "inlineStr":
		if c.InlineString == nil {
			return ""
		}
		return c.InlineString.String()
	default:
		return strings.TrimSpace(c.Value)
	}
}

type sharedStringItem struct {
	Text string            `xml:"t"`
	Runs []sharedStringRun `xml:"r"`
}

type sharedStringRun struct {
	Text string `xml:"t"`
}

func (s sharedStringItem) String() string {
	if s.Text != "" {
		return s.Text
	}
	var builder strings.Builder
	for _, run := range s.Runs {
		builder.WriteString(run.Text)
	}
	return builder.String()
}

func describeWordSource(path string) string {
	return fmt.Sprintf("imported from %s", filepath.Base(path))
}
