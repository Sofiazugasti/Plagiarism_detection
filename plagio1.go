package main

import (
	"bufio"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Estructura para almacenar los resultados de la comparación
type resultado struct {
	File1, File2       string
	Content1, Content2 template.HTML
	Similitud          float64
	CommonSubstrings   []string
}

// Leer archivos desde una carpeta
func readFilesFromFolder(folder string) ([]string, error) {
	var fileList []string
	files, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".txt" {
			fileList = append(fileList, filepath.Join(folder, file.Name()))
		}
	}
	return fileList, nil
}

// Leer el contenido de un archivo
func readFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text() + " ")
	}
	return content.String(), scanner.Err()
}

// Función LCS para encontrar la subcadena común más larga
func lcs(text1, text2 string) string {
	n, m := len(text1), len(text2)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	length := 0
	endPos := 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if text1[i-1] == text2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
				if dp[i][j] > length {
					length = dp[i][j]
					endPos = i
				}
			}
		}
	}

	if length == 0 {
		return ""
	}
	return text1[endPos-length : endPos]
}

// Encontrar todas las subcadenas comunes
func findAllLCS(text1, text2 string, minLength int) []string {
	var commonSubstrings []string
	stopWords := map[string]bool{
		"the": true, "is": true, "in": true, "it": true, "and": true, "to": true, "of": true, "a": true, "that": true, "I": true, "you": true, "he": true, "she": true, "they": true, "we": true, "be": true, "have": true, "has": true, "with": true, "for": true, "on": true, "at": true, "by": true, "this": true, "from": true, "or": true, "as": true, "an": true, "if": true,
	}

	for {
		lcsStr := lcs(text1, text2)
		if len(lcsStr) < minLength {
			break
		}

		words := strings.Fields(lcsStr)
		allStopWords := true
		for _, word := range words {
			if !stopWords[strings.ToLower(word)] {
				allStopWords = false
				break
			}
		}

		if !allStopWords {
			commonSubstrings = append(commonSubstrings, lcsStr)
		}

		text1 = removeSubstring(text1, lcsStr)
		text2 = removeSubstring(text2, lcsStr)
	}

	return commonSubstrings
}

// Resaltar subcadenas comunes en el texto y devolver HTML seguro
func highlightCommonSubstrings(text string, substrings []string) template.HTML {
	for _, substring := range substrings {
		text = strings.ReplaceAll(text, substring, "<mark>"+substring+"</mark>")
	}
	return template.HTML(text) // Marcar explícitamente como HTML seguro
}

// Eliminar una subcadena de un texto
func removeSubstring(text, substring string) string {
	return strings.Replace(text, substring, "", -1)
}

// Calcular el porcentaje de similitud y subcadenas comunes
func calcularSimilitud(text1, text2 string, minLength int) (float64, []string) {
	commonSubstrings := findAllLCS(text1, text2, minLength)

	totalLengthCommon := 0
	for _, sub := range commonSubstrings {
		totalLengthCommon += len(sub)
	}

	totalLength := (len(text1) + len(text2)) / 2
	similitud := math.Round((float64(totalLengthCommon) / float64(totalLength)) * 100)

	return similitud, commonSubstrings
}

// Comparar archivos y generar resultados en HTML
func compararArchivosHTML(folder string, minLength int) ([]resultado, error) {
	fileList, err := readFilesFromFolder(folder)
	if err != nil {
		return nil, err
	}

	textos := make([]string, len(fileList))
	for i, file := range fileList {
		content, err := readFile(file)
		if err != nil {
			return nil, err
		}
		textos[i] = content
	}

	var resultados []resultado
	for i := 0; i < len(fileList)-1; i++ {
		for j := i + 1; j < len(fileList); j++ {
			similitud, commonSubstrings := calcularSimilitud(textos[i], textos[j], minLength)
			content1 := highlightCommonSubstrings(textos[i], commonSubstrings)
			content2 := highlightCommonSubstrings(textos[j], commonSubstrings)
			resultados = append(resultados, resultado{fileList[i], fileList[j], content1, content2, similitud, commonSubstrings})
		}
	}

	sort.Slice(resultados, func(i, j int) bool {
		return resultados[i].Similitud > resultados[j].Similitud
	})

	return resultados[:10], nil
}

// Handler para mostrar el formulario y los resultados
func handler(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "pregunta_a" // Carpeta por defecto si no se proporciona una
	}

	resultados, err := compararArchivosHTML(folder, 5)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	funcMap := template.FuncMap{"addOne": func(i int) int { return i + 1 }}

	tmpl := template.Must(template.New("results").Funcs(funcMap).Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Detector de plagio para docentes Tec</title>
    <style>
        body {
            background-color: #87CEEB; 
            font-family: Arial, sans-serif; 
            color: #333;
            margin: 0;
            padding: 20px;
        }
        h1 {
            font-family: Arial, sans-serif;
            color: #222;
        }
        .content {
            margin: 10px;
            padding: 10px;
            background-color: #f8f8f8;
            border-left: 3px solid #f8f8f8;
            white-space: pre-wrap;
            overflow-wrap: break-word;
        }
        mark {
            background-color: #fff068;
            font-weight: bold;
        }
		
		.mb {
			padding: 10px 20px;
			background-color: #278891;
			color: 	white;
			border: 1.5px solid  #333;
			border-radius: 5px;
			cursor: pointer;
			width: 170px;
		}
		.mt {
			padding: 10px 20px;
			background-color: #white;
			color: #333;
			border: 1.5px solid  #333;
			border-radius: 5px;
			cursor: pointer;

		}
		.mb:hover {
  			background-color: #808080;
		}

    </style>
</head>
<body>
    <h1>Detector de plagio para docentes Tec</h1>
    <form method="get">
        <label for="folder">Ingresa el nombre de la carpeta:</label>
        <input class="mt" id="folder" name="folder" value="{{.Folder}}">
        <button class="mb" >Analizar carpeta</button>
    </form>
    <h2>Top 10 Archivos con Mayor Similitud en la Carpeta '{{.Folder}}'</h2>
    {{range $i, $res := .Resultados}}
        <h2> El archivo: {{addOne $i}}. {{$res.File1}} y el archivo: {{$res.File2}}: {{$res.Similitud}}% de similitud</h2>
        <p><strong> Contenido del archivo: {{$res.File1}}:</strong></p>
        <div class="content">{{ $res.Content1 }}</div>
        <p><strong> Contenido del archivo: {{$res.File2}}:</strong></p>
        <div class="content">{{ $res.Content2 }}</div>
    {{end}}
</body>
</html>
`))

	data := struct {
		Folder     string
		Resultados []resultado
	}{
		Folder:     folder,
		Resultados: resultados,
	}

	if err := tmpl.Execute(w, data); err != nil {
		fmt.Fprintf(w, "Error rendering template: %v", err)
	}
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Servidor iniciado en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
