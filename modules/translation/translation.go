// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package translation

import (
	"context"
	"html/template"
	"sort"
	"strings"
	"sync"

	"forgejo.org/modules/log"
	"forgejo.org/modules/options"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/translation/i18n"
	"forgejo.org/modules/util"

	"github.com/dustin/go-humanize"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

type contextKey struct{}

var ContextKey any = &contextKey{}

// Locale represents an interface to translation
//
// If this gets modified, remember to also adjust
// build/lint-locale-usage/lint-locale-usage.go's InitLocaleTrFunctions(),
// which requires to know in what argument positions `trKey`'s are given.
type Locale interface {
	Language() string
	TrString(string, ...any) string

	Tr(key string, args ...any) template.HTML
	// New-style pluralized strings
	TrPluralString(count any, trKey string, trArgs ...any) template.HTML
	// Old-style pseudo-pluralized strings, deprecated
	TrN(cnt any, key1, keyN string, args ...any) template.HTML

	TrSize(size int64) ReadableSize

	HasKey(trKey string) bool

	PrettyNumber(v any) string

	TrPluralStringAllForms(trKey string) ([]string, []string)
}

// LangType represents a lang type
type LangType struct {
	Lang, Name string // these fields are used directly in templates: {{range .AllLangs}}{{.Lang}}{{.Name}}{{end}}
}

var (
	lock *sync.RWMutex

	allLangs   []*LangType
	allLangMap map[string]*LangType

	matcher       language.Matcher
	supportedTags []language.Tag
)

// AllLangs returns all supported languages sorted by name
func AllLangs() []*LangType {
	return allLangs
}

// InitLocales loads the locales
func InitLocales(ctx context.Context) {
	if lock != nil {
		lock.Lock()
		defer lock.Unlock()
	} else if !setting.IsProd && lock == nil {
		lock = &sync.RWMutex{}
	}

	refreshLocales := func() {
		i18n.ResetDefaultLocales()
		localeNames, err := options.AssetFS().ListFiles("locale", true)
		if err != nil {
			log.Fatal("Failed to list locale files: %v", err)
		}

		localeData := make(map[string][]byte, len(localeNames))
		for _, name := range localeNames {
			localeData[name], err = options.Locale(name)
			if err != nil {
				log.Fatal("Failed to load %s locale file. %v", name, err)
			}
		}

		supportedTags = make([]language.Tag, len(setting.Langs))
		for i, lang := range setting.Langs {
			supportedTags[i] = language.Raw.Make(lang)
		}

		matcher = language.NewMatcher(supportedTags)
		for i := range setting.Names {
			var localeDataBase []byte
			if i == 0 && setting.Langs[0] != "en-US" {
				// Only en-US has complete translations. When use other language as default, the en-US should still be used as fallback.
				localeDataBase = localeData["locale_en-US.ini"]
				if localeDataBase == nil {
					log.Fatal("Failed to load locale_en-US.ini file.")
				}
			}

			pluralRuleIndex := GetPluralRuleImpl(setting.Langs[i])
			key := "locale_" + setting.Langs[i] + ".ini"
			if err = i18n.DefaultLocales.AddLocaleByIni(setting.Langs[i], setting.Names[i], PluralRules[pluralRuleIndex], UsedPluralForms[pluralRuleIndex], localeDataBase, localeData[key]); err != nil {
				log.Error("Failed to set old-style messages to %s: %v", setting.Langs[i], err)
			}

			key = "locale_next/locale_" + setting.Langs[i] + ".json"
			if bytes, err := options.AssetFS().ReadFile(key); err == nil {
				if err = i18n.DefaultLocales.AddToLocaleFromJSON(setting.Langs[i], bytes); err != nil {
					log.Error("Failed to add new-style messages to %s: %v", setting.Langs[i], err)
				}
			} else {
				log.Error("Failed to open new-style messages for %s: %v", setting.Langs[i], err)
			}
		}
		if len(setting.Langs) != 0 {
			defaultLangName := setting.Langs[0]
			if defaultLangName != "en-US" {
				log.Info("Use the first locale (%s) in LANGS setting option as default", defaultLangName)
			}
			i18n.DefaultLocales.SetDefaultLang(defaultLangName)
		}
	}

	refreshLocales()

	langs, descs := i18n.DefaultLocales.ListLangNameDesc()
	allLangs = make([]*LangType, 0, len(langs))
	allLangMap = map[string]*LangType{}
	for i, v := range langs {
		l := &LangType{v, descs[i]}
		allLangs = append(allLangs, l)
		allLangMap[v] = l
	}

	// Sort languages case-insensitive according to their name - needed for the user settings
	sort.Slice(allLangs, func(i, j int) bool {
		return strings.ToLower(allLangs[i].Name) < strings.ToLower(allLangs[j].Name)
	})

	if !setting.IsProd {
		go options.AssetFS().WatchLocalChanges(ctx, func() {
			lock.Lock()
			defer lock.Unlock()
			refreshLocales()
		})
	}
}

// Match matches accept languages
func Match(tags ...language.Tag) language.Tag {
	_, i, _ := matcher.Match(tags...)
	return supportedTags[i]
}

// locale represents the information of localization.
type locale struct {
	i18n.Locale
	Lang, LangName string // these fields are used directly in templates: ctx.Locale.Lang
	msgPrinter     *message.Printer
}

var _ Locale = (*locale)(nil)

// NewLocale return a locale
func NewLocale(lang string) Locale {
	if lock != nil {
		lock.RLock()
		defer lock.RUnlock()
	}

	if lang == "dummy" {
		l := &locale{
			Locale:     &i18n.KeyLocale{},
			Lang:       lang,
			LangName:   lang,
			msgPrinter: message.NewPrinter(language.English),
		}
		return l
	}

	langName := "unknown"
	if l, ok := allLangMap[lang]; ok {
		langName = l.Name
	} else if len(setting.Langs) > 0 {
		lang = setting.Langs[0]
		langName = setting.Names[0]
	}

	i18nLocale, _ := i18n.GetLocale(lang)
	l := &locale{
		Locale:   i18nLocale,
		Lang:     lang,
		LangName: langName,
	}
	if langTag, err := language.Parse(lang); err != nil {
		log.Error("Failed to parse language tag from name %q: %v", l.Lang, err)
		l.msgPrinter = message.NewPrinter(language.English)
	} else {
		l.msgPrinter = message.NewPrinter(langTag)
	}
	return l
}

func (l *locale) Language() string {
	return l.Lang
}

// Language specific rules for translating plural texts
var trNLangRules = map[string]func(int64) int{
	// the default rule is "en-US" if a language isn't listed here
	"en-US": func(cnt int64) int {
		if cnt == 1 {
			return 0
		}
		return 1
	},
	"lv-LV": func(cnt int64) int {
		if cnt%10 == 1 && cnt%100 != 11 {
			return 0
		}
		return 1
	},
	"ru-RU": func(cnt int64) int {
		if cnt%10 == 1 && cnt%100 != 11 {
			return 0
		}
		return 1
	},
	"zh-CN": func(cnt int64) int {
		return 0
	},
	"zh-HK": func(cnt int64) int {
		return 0
	},
	"zh-TW": func(cnt int64) int {
		return 0
	},
	"fr-FR": func(cnt int64) int {
		if cnt > -2 && cnt < 2 {
			return 0
		}
		return 1
	},
}

func (l *locale) Tr(s string, args ...any) template.HTML {
	return l.TrHTML(s, args...)
}

// TrN returns translated message for plural text translation
func (l *locale) TrN(cnt any, key1, keyN string, args ...any) template.HTML {
	var c int64
	if t, ok := cnt.(int); ok {
		c = int64(t)
	} else if t, ok := cnt.(int16); ok {
		c = int64(t)
	} else if t, ok := cnt.(int32); ok {
		c = int64(t)
	} else if t, ok := cnt.(int64); ok {
		c = t
	} else {
		return l.Tr(keyN, args...)
	}

	ruleFunc, ok := trNLangRules[l.Lang]
	if !ok {
		ruleFunc = trNLangRules["en-US"]
	}

	if ruleFunc(c) == 0 {
		return l.Tr(key1, args...)
	}
	return l.Tr(keyN, args...)
}

type ReadableSize struct {
	PrettyNumber   string
	TranslatedUnit string
}

func (bs ReadableSize) String() string {
	return bs.PrettyNumber + " " + bs.TranslatedUnit
}

// TrSize returns array containing pretty formatted size and localized output of FileSize
// output of humanize.IBytes has to be split in order to be localized
func (l *locale) TrSize(s int64) ReadableSize {
	us := uint64(s)
	if s < 0 {
		us = uint64(-s)
	}
	untranslated := humanize.IBytes(us)
	if s < 0 {
		untranslated = "-" + untranslated
	}
	numberVal, unitVal, found := strings.Cut(untranslated, " ")
	if !found {
		log.Error("no space in go-humanized size of %d: %q", s, untranslated)
	}
	numberVal = l.PrettyNumber(numberVal)
	unitVal = l.TrString("munits.data." + strings.ToLower(unitVal))
	return ReadableSize{numberVal, unitVal}
}

func (l *locale) PrettyNumber(v any) string {
	// TODO: this mechanism is not good enough, the complete solution is to switch the translation system to ICU message format
	if s, ok := v.(string); ok {
		if num, err := util.ToInt64(s); err == nil {
			v = num
		} else if num, err := util.ToFloat64(s); err == nil {
			v = num
		}
	}
	return l.msgPrinter.Sprintf("%v", number.Decimal(v))
}

func GetPluralRule(l Locale) int {
	return GetPluralRuleImpl(l.Language())
}

func GetDefaultPluralRule() int {
	return GetPluralRuleImpl(i18n.DefaultLocales.GetDefaultLang())
}

func init() {
	// prepare a default matcher, especially for tests
	supportedTags = []language.Tag{language.English}
	matcher = language.NewMatcher(supportedTags)
}
