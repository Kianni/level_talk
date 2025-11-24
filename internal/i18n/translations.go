package i18n

import (
	"html/template"
)

// Supported languages
const (
	LangEN = "en"
	LangFI = "fi"
	LangSV = "sv"
	LangRU = "ru"
	LangES = "es"
	LangJA = "ja"
	LangDE = "de"
)

// DefaultLanguage is the fallback language
const DefaultLanguage = LangEN

// LanguageNames maps language codes to their display names
var LanguageNames = map[string]string{
	LangEN: "English",
	LangFI: "Suomi",
	LangSV: "Svenska",
	LangRU: "Русский",
	LangES: "Español",
	LangJA: "日本語",
	LangDE: "Deutsch",
}

// Translations holds all translations
type Translations map[string]map[string]string

// GetTranslations returns all translation maps
func GetTranslations() Translations {
	return translations
}

// Get returns a translation for a given language and key
func Get(lang, key string) string {
	if trans, ok := translations[lang]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	// Fallback to English
	if trans, ok := translations[DefaultLanguage]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	return key
}

// T returns a template function for translations
func T(lang string) func(string) template.HTML {
	return func(key string) template.HTML {
		return template.HTML(Get(lang, key))
	}
}

var translations = Translations{
	LangEN: {
		"app_name":           "LevelTalk",
		"create_dialog":      "Create a dialog",
		"input_language":     "Input language",
		"dialog_language":    "Dialog language",
		"cefr_level":         "CEFR level",
		"words_phrases":      "Words / phrases",
		"generate_dialog":    "Generate dialog",
		"search_dialogs":     "Search saved dialogs",
		"any":                "Any",
		"filter":             "Filter",
		"dialogs":            "Dialogs",
		"latest_results":     "Latest results",
		"no_dialogs":         "No dialogs yet. Generate one!",
		"name":               "Name",
		"input":              "Input",
		"dialog":             "Dialog",
		"created":            "Created",
		"open":               "Open",
		"back":               "Back",
		"download_selected_text":   "Download Selected Text",
		"download_selected_audio": "Download Selected Audio",
		"select_all":         "Select all",
		"vocabulary":          "Vocabulary",
		"words_from":          "Words from",
		"translation_in_dialog": "(translation in dialog)",
		"language":            "Language",
		"input_language_label": "Input language",
		"dialog_language_label": "Dialog language",
		"cefr":                "CEFR",
		"tagline":             "Make your vocabulary speak — in any language, at any level.",
	},
	LangFI: {
		"app_name":           "LevelTalk",
		"create_dialog":      "Luo vuoropuhelu",
		"input_language":     "Syötteen kieli",
		"dialog_language":    "Vuoropuhelun kieli",
		"cefr_level":         "CEFR-taso",
		"words_phrases":      "Sanat / lauseet",
		"generate_dialog":    "Luo vuoropuhelu",
		"search_dialogs":     "Etsi tallennettuja vuoropuheluja",
		"any":                "Mikä tahansa",
		"filter":             "Suodata",
		"dialogs":            "Vuoropuhelut",
		"latest_results":     "Uusimmat tulokset",
		"no_dialogs":         "Ei vuoropuheluja vielä. Luo yksi!",
		"name":               "Nimi",
		"input":              "Syöte",
		"dialog":             "Vuoropuhelu",
		"created":            "Luotu",
		"open":               "Avaa",
		"back":               "Takaisin",
		"download_selected_text":   "Lataa valitut tekstit",
		"download_selected_audio": "Lataa valitut äänet",
		"select_all":         "Valitse kaikki",
		"vocabulary":          "Sanasto",
		"words_from":          "Sanat",
		"translation_in_dialog": "(käännös vuoropuhelussa)",
		"language":            "Kieli",
		"input_language_label": "Syötteen kieli",
		"dialog_language_label": "Vuoropuhelun kieli",
		"cefr":                "CEFR",
		"tagline":             "Anna sanastollesi ääni — millä tahansa kielellä, millä tahansa tasolla.",
	},
	LangSV: {
		"app_name":           "LevelTalk",
		"create_dialog":      "Skapa en dialog",
		"input_language":     "Ingångsspråk",
		"dialog_language":    "Dialogspråk",
		"cefr_level":         "CEFR-nivå",
		"words_phrases":      "Ord / fraser",
		"generate_dialog":    "Generera dialog",
		"search_dialogs":     "Sök sparade dialoger",
		"any":                "Valfritt",
		"filter":             "Filtrera",
		"dialogs":            "Dialoger",
		"latest_results":     "Senaste resultat",
		"no_dialogs":         "Inga dialoger ännu. Skapa en!",
		"name":               "Namn",
		"input":              "Ingång",
		"dialog":             "Dialog",
		"created":            "Skapad",
		"open":               "Öppna",
		"back":               "Tillbaka",
		"download_selected_text":   "Ladda ner valda texter",
		"download_selected_audio": "Ladda ner valda ljud",
		"select_all":         "Välj alla",
		"vocabulary":          "Ordförråd",
		"words_from":          "Ord från",
		"translation_in_dialog": "(översättning i dialogen)",
		"language":            "Språk",
		"input_language_label": "Ingångsspråk",
		"dialog_language_label": "Dialogspråk",
		"cefr":                "CEFR",
		"tagline":             "Låt ditt ordförråd tala — på vilket språk som helst, på vilken nivå som helst.",
	},
	LangRU: {
		"app_name":           "LevelTalk",
		"create_dialog":      "Создать диалог",
		"input_language":     "Язык ввода",
		"dialog_language":    "Язык диалога",
		"cefr_level":         "Уровень CEFR",
		"words_phrases":      "Слова / фразы",
		"generate_dialog":    "Создать диалог",
		"search_dialogs":     "Поиск сохранённых диалогов",
		"any":                "Любой",
		"filter":             "Фильтр",
		"dialogs":            "Диалоги",
		"latest_results":     "Последние результаты",
		"no_dialogs":         "Пока нет диалогов. Создайте один!",
		"name":               "Название",
		"input":              "Ввод",
		"dialog":             "Диалог",
		"created":            "Создан",
		"open":               "Открыть",
		"back":               "Назад",
		"download_selected_text":   "Скачать выбранные тексты",
		"download_selected_audio": "Скачать выбранные аудио",
		"select_all":         "Выбрать всё",
		"vocabulary":          "Словарь",
		"words_from":          "Слова из",
		"translation_in_dialog": "(перевод в диалоге)",
		"language":            "Язык",
		"input_language_label": "Язык ввода",
		"dialog_language_label": "Язык диалога",
		"cefr":                "CEFR",
		"tagline":             "Заставьте свой словарный запас говорить — на любом языке, на любом уровне.",
	},
	LangES: {
		"app_name":           "LevelTalk",
		"create_dialog":      "Crear un diálogo",
		"input_language":     "Idioma de entrada",
		"dialog_language":    "Idioma del diálogo",
		"cefr_level":         "Nivel CEFR",
		"words_phrases":      "Palabras / frases",
		"generate_dialog":    "Generar diálogo",
		"search_dialogs":     "Buscar diálogos guardados",
		"any":                "Cualquiera",
		"filter":             "Filtrar",
		"dialogs":            "Diálogos",
		"latest_results":     "Últimos resultados",
		"no_dialogs":         "Aún no hay diálogos. ¡Crea uno!",
		"name":               "Nombre",
		"input":              "Entrada",
		"dialog":             "Diálogo",
		"created":            "Creado",
		"open":               "Abrir",
		"back":               "Atrás",
		"download_selected_text":   "Descargar textos seleccionados",
		"download_selected_audio": "Descargar audios seleccionados",
		"select_all":         "Seleccionar todo",
		"vocabulary":          "Vocabulario",
		"words_from":          "Palabras de",
		"translation_in_dialog": "(traducción en el diálogo)",
		"language":            "Idioma",
		"input_language_label": "Idioma de entrada",
		"dialog_language_label": "Idioma del diálogo",
		"cefr":                "CEFR",
		"tagline":             "Haz que tu vocabulario hable — en cualquier idioma, en cualquier nivel.",
	},
	LangJA: {
		"app_name":           "LevelTalk",
		"create_dialog":      "対話を作成",
		"input_language":     "入力言語",
		"dialog_language":    "対話言語",
		"cefr_level":         "CEFRレベル",
		"words_phrases":      "単語 / フレーズ",
		"generate_dialog":    "対話を生成",
		"search_dialogs":     "保存された対話を検索",
		"any":                "任意",
		"filter":             "フィルター",
		"dialogs":            "対話",
		"latest_results":     "最新の結果",
		"no_dialogs":         "対話はまだありません。作成してください！",
		"name":               "名前",
		"input":              "入力",
		"dialog":             "対話",
		"created":            "作成日",
		"open":               "開く",
		"back":               "戻る",
		"download_selected_text":   "選択したテキストをダウンロード",
		"download_selected_audio": "選択した音声をダウンロード",
		"select_all":         "すべて選択",
		"vocabulary":          "語彙",
		"words_from":          "単語",
		"translation_in_dialog": "(対話内の翻訳)",
		"language":            "言語",
		"input_language_label": "入力言語",
		"dialog_language_label": "対話言語",
		"cefr":                "CEFR",
		"tagline":             "語彙に声を与える — あらゆる言語で、あらゆるレベルで。",
	},
	LangDE: {
		"app_name":           "LevelTalk",
		"create_dialog":      "Dialog erstellen",
		"input_language":     "Eingabesprache",
		"dialog_language":    "Dialogsprache",
		"cefr_level":         "CEFR-Niveau",
		"words_phrases":      "Wörter / Phrasen",
		"generate_dialog":    "Dialog generieren",
		"search_dialogs":     "Gespeicherte Dialoge suchen",
		"any":                "Beliebig",
		"filter":             "Filtern",
		"dialogs":            "Dialoge",
		"latest_results":     "Neueste Ergebnisse",
		"no_dialogs":         "Noch keine Dialoge. Erstellen Sie einen!",
		"name":               "Name",
		"input":              "Eingabe",
		"dialog":             "Dialog",
		"created":            "Erstellt",
		"open":               "Öffnen",
		"back":               "Zurück",
		"download_selected_text":   "Ausgewählte Texte herunterladen",
		"download_selected_audio": "Ausgewählte Audios herunterladen",
		"select_all":         "Alle auswählen",
		"vocabulary":          "Wortschatz",
		"words_from":          "Wörter von",
		"translation_in_dialog": "(Übersetzung im Dialog)",
		"language":            "Sprache",
		"input_language_label": "Eingabesprache",
		"dialog_language_label": "Dialogsprache",
		"cefr":                "CEFR",
		"tagline":             "Lassen Sie Ihren Wortschatz sprechen — in jeder Sprache, auf jedem Niveau.",
	},
}

