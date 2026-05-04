package catalog

import (
	"archive/zip"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type KnowledgeBaseChunk struct {
	Title   string
	Content string
}

func LoadKnowledgeBaseChunksFromFile(path, fallbackTitle string) ([]KnowledgeBaseChunk, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	switch ext {
	case ".txt", ".md":
		return loadKnowledgeBaseText(path, fallbackTitle)
	case ".csv":
		return loadKnowledgeBaseCSV(path, fallbackTitle)
	case ".xlsx":
		return loadKnowledgeBaseXLSX(path, fallbackTitle)
	case ".docx":
		return loadKnowledgeBaseDOCX(path, fallbackTitle)
	default:
		return nil, errors.New("unsupported knowledge base file type")
	}
}

func loadKnowledgeBaseText(path, fallbackTitle string) ([]KnowledgeBaseChunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := cleanKnowledgeBaseText(string(data))
	if text == "" {
		return nil, errors.New("knowledge base file is empty")
	}
	return chunkKnowledgeBaseText(firstNonBlankTitle(fallbackTitle, filepath.Base(path)), text), nil
}

func loadKnowledgeBaseCSV(path, fallbackTitle string) ([]KnowledgeBaseChunk, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("knowledge base file is empty")
	}

	title := firstNonBlankTitle(fallbackTitle, filepath.Base(path))
	var chunks []KnowledgeBaseChunk
	for index, row := range rows {
		values := make([]string, 0, len(row))
		for _, cell := range row {
			cell = cleanKnowledgeBaseText(cell)
			if cell != "" {
				values = append(values, cell)
			}
		}
		if len(values) == 0 {
			continue
		}
		content := strings.Join(values, " | ")
		if index == 0 && looksLikeHeaderRow(values) {
			content = "表头: " + content
		}
		chunks = append(chunks, KnowledgeBaseChunk{
			Title:   title,
			Content: content,
		})
	}
	if len(chunks) == 0 {
		return nil, errors.New("knowledge base file is empty")
	}
	return chunks, nil
}

func loadKnowledgeBaseXLSX(path, fallbackTitle string) ([]KnowledgeBaseChunk, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	sharedStrings, err := readSharedStrings(reader.File)
	if err != nil {
		return nil, err
	}

	worksheetNames := make([]string, 0, len(reader.File))
	lookup := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") {
			worksheetNames = append(worksheetNames, file.Name)
			lookup[file.Name] = file
		}
	}
	sort.Strings(worksheetNames)
	if len(worksheetNames) == 0 {
		return nil, errors.New("worksheet not found")
	}

	title := firstNonBlankTitle(fallbackTitle, filepath.Base(path))
	chunks := make([]KnowledgeBaseChunk, 0, 128)
	for _, worksheetName := range worksheetNames {
		file := lookup[worksheetName]
		if file == nil {
			continue
		}
		rows, err := readWorksheetRows(file, sharedStrings)
		if err != nil {
			return nil, err
		}
		for index, row := range rows {
			values := make([]string, 0, len(row))
			for _, cell := range row {
				cell = cleanKnowledgeBaseText(cell)
				if cell != "" {
					values = append(values, cell)
				}
			}
			if len(values) == 0 {
				continue
			}
			content := strings.Join(values, " | ")
			if index == 0 && looksLikeHeaderRow(values) {
				content = "表头: " + content
			}
			chunks = append(chunks, KnowledgeBaseChunk{
				Title:   title,
				Content: content,
			})
		}
	}
	if len(chunks) == 0 {
		return nil, errors.New("knowledge base file is empty")
	}
	return chunks, nil
}

func loadKnowledgeBaseDOCX(path, fallbackTitle string) ([]KnowledgeBaseChunk, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var documentFile *zip.File
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}
	if documentFile == nil {
		return nil, errors.New("word document content not found")
	}

	paragraphs, err := readDOCXParagraphs(documentFile)
	if err != nil {
		return nil, err
	}

	parts := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = cleanKnowledgeBaseText(paragraph)
		if paragraph != "" {
			parts = append(parts, paragraph)
		}
	}
	if len(parts) == 0 {
		return nil, errors.New("knowledge base file is empty")
	}

	text := strings.Join(parts, "\n")
	return chunkKnowledgeBaseText(firstNonBlankTitle(fallbackTitle, filepath.Base(path)), text), nil
}

func readDOCXParagraphs(documentFile *zip.File) ([]string, error) {
	rc, err := documentFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	paragraphs := make([]string, 0, 64)
	var builder strings.Builder
	inText := false

	flushParagraph := func() {
		paragraph := builder.String()
		builder.Reset()
		if strings.TrimSpace(paragraph) != "" {
			paragraphs = append(paragraphs, paragraph)
		}
	}

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "t":
				inText = true
			case "tab":
				builder.WriteString("\t")
			case "br", "cr":
				builder.WriteString("\n")
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				flushParagraph()
			}
		case xml.CharData:
			if inText {
				builder.WriteString(string(value))
			}
		}
	}

	flushParagraph()
	return paragraphs, nil
}

func readWorksheetRows(sheetFile *zip.File, sharedStrings []string) ([][]string, error) {
	rc, err := sheetFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var rows [][]string
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

		rowValues := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			value := strings.TrimSpace(cell.Resolve(sharedStrings))
			rowValues = append(rowValues, value)
		}
		rows = append(rows, rowValues)
	}
	return rows, nil
}

func chunkKnowledgeBaseText(title, text string) []KnowledgeBaseChunk {
	const maxRunes = 900
	const overlapRunes = 120

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return []KnowledgeBaseChunk{{
			Title:   title,
			Content: text,
		}}
	}

	chunks := make([]KnowledgeBaseChunk, 0, (len(runes)/maxRunes)+1)
	for start := 0; start < len(runes); {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}

		if end < len(runes) {
			if split := findKnowledgeBaseSplit(runes, start, end); split > start {
				end = split
			}
		}

		content := cleanKnowledgeBaseText(string(runes[start:end]))
		if content != "" {
			chunks = append(chunks, KnowledgeBaseChunk{
				Title:   title,
				Content: content,
			})
		}

		if end >= len(runes) {
			break
		}
		nextStart := end - overlapRunes
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	return chunks
}

func findKnowledgeBaseSplit(runes []rune, start, end int) int {
	min := start + (end-start)/2
	for i := end - 1; i >= min; i-- {
		switch runes[i] {
		case '\n', '\r', '。', '！', '？', '.', '!', '?', ';', '；':
			return i + 1
		}
	}
	return end
}

func looksLikeHeaderRow(values []string) bool {
	if len(values) == 0 {
		return false
	}
	headerHits := 0
	for _, value := range values {
		normalized := normalizeHeader(value)
		switch normalized {
		case "id", "title", "name", "content", "text", "question", "answer", "category", "标签", "标题", "内容":
			headerHits++
			continue
		}
		if len([]rune(value)) <= 12 && allHeaderLike(value) {
			headerHits++
		}
	}
	return headerHits > 0 && headerHits*2 >= len(values)
}

func allHeaderLike(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) {
			continue
		}
		if r == '_' || r == '-' || r == ' ' {
			continue
		}
		return false
	}
	return true
}

func cleanKnowledgeBaseText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	lastBlank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if lastBlank {
				continue
			}
			lastBlank = true
			cleaned = append(cleaned, "")
			continue
		}
		lastBlank = false
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func firstNonBlankTitle(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return "知识库文档"
}
