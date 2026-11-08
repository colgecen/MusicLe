package components

import (
	"MusicLe/state"
	"MusicLe/ui"
)

// navLabels holds every header tab label in all 11 languages, without
// surrounding spaces (spacing comes from the tab style padding, so it is
// counted exactly once).
var navLabels = map[string]map[state.Language]string{
	"home": {
		state.LangTurkish:    "Ana Sayfa",
		state.LangEnglish:    "Home",
		state.LangSpanish:    "Inicio",
		state.LangGerman:     "Startseite",
		state.LangFrench:     "Accueil",
		state.LangArabic:     "الرئيسية",
		state.LangPortuguese: "Início",
		state.LangChinese:    "首页",
		state.LangJapanese:   "ホーム",
		state.LangItalian:    "Home",
		state.LangRussian:    "Главная",
	},
	"downloads": {
		state.LangTurkish:    "İndirilenler",
		state.LangEnglish:    "Downloads",
		state.LangSpanish:    "Descargas",
		state.LangGerman:     "Downloads",
		state.LangFrench:     "Téléchargements",
		state.LangArabic:     "التنزيلات",
		state.LangPortuguese: "Downloads",
		state.LangChinese:    "下载",
		state.LangJapanese:   "ダウンロード",
		state.LangItalian:    "Download",
		state.LangRussian:    "Загрузки",
	},
	"profile": {
		state.LangTurkish:    "Profil",
		state.LangEnglish:    "Profile",
		state.LangSpanish:    "Perfil",
		state.LangGerman:     "Profil",
		state.LangFrench:     "Profil",
		state.LangArabic:     "الملف الشخصي",
		state.LangPortuguese: "Perfil",
		state.LangChinese:    "个人资料",
		state.LangJapanese:   "プロフィール",
		state.LangItalian:    "Profilo",
		state.LangRussian:    "Профиль",
	},
	"playlist": {
		state.LangTurkish:    "Çalma Listesi",
		state.LangEnglish:    "Playlist",
		state.LangSpanish:    "Lista",
		state.LangGerman:     "Playlist",
		state.LangFrench:     "Playlist",
		state.LangArabic:     "قائمة التشغيل",
		state.LangPortuguese: "Playlist",
		state.LangChinese:    "播放列表",
		state.LangJapanese:   "プレイリスト",
		state.LangItalian:    "Playlist",
		state.LangRussian:    "Плейлист",
	},
	"settings": {
		state.LangTurkish:    "Genel",
		state.LangEnglish:    "General",
		state.LangSpanish:    "General",
		state.LangGerman:     "Allgemein",
		state.LangFrench:     "Général",
		state.LangArabic:     "عام",
		state.LangPortuguese: "Geral",
		state.LangChinese:    "通用",
		state.LangJapanese:   "一般",
		state.LangItalian:    "Generali",
		state.LangRussian:    "Общие",
	},
}

// navTabIDs lists the header tabs in display order.
var navTabIDs = []string{"home", "downloads", "profile", "playlist", "settings"}

// navTabPadH mirrors the horizontal padding of the tab style in header.go.
const navTabPadH = 2

// NavLabel returns the clean (unpadded) tab label for the current language.
func NavLabel(id string) string {
	if m, ok := navLabels[id]; ok {
		if s, ok := m[state.Current.Language]; ok {
			return s
		}
		if s, ok := m[state.LangEnglish]; ok {
			return s
		}
	}
	return id
}

// NavTabWidth returns the fixed content width for a nav tab: the longest
// label across all 11 languages plus horizontal padding. The value never
// depends on the active language, so tabs keep their exact size and position
// when the language changes.
func NavTabWidth(id string) int {
	max := 0
	for _, s := range navLabels[id] {
		if w := ui.TextWidth(s); w > max {
			max = w
		}
	}
	return max + navTabPadH*2
}

// noTrackText returns the localized "No track playing" string.
func noTrackText() string {
	switch state.Current.Language {
	case state.LangTurkish:
		return "Çalan parça yok"
	case state.LangSpanish:
		return "Ninguna canción en reproducción"
	case state.LangGerman:
		return "Kein Titel läuft"
	case state.LangFrench:
		return "Aucun morceau en lecture"
	case state.LangArabic:
		return "لا يوجد مقطع قيد التشغيل"
	case state.LangPortuguese:
		return "Nenhuma faixa tocando"
	case state.LangChinese:
		return "当前没有播放"
	case state.LangJapanese:
		return "再生中の曲なし"
	case state.LangItalian:
		return "Nessun brano in riproduzione"
	case state.LangRussian:
		return "Ничего не играет"
	default:
		return "No track playing"
	}
}

// volLabel returns the short volume caption for the player bar.
func volLabel() string {
	switch state.Current.Language {
	case state.LangTurkish:
		return "SES"
	case state.LangGerman:
		return "LAUT"
	case state.LangArabic:
		return "صوت"
	case state.LangChinese:
		return "音量"
	case state.LangJapanese:
		return "音量"
	case state.LangRussian:
		return "ЗВУК"
	default:
		return "VOL"
	}
}

// langBadge returns the uppercase language code (EN, TR, ES, DE, FR, AR,
// PT, ZH, JA, IT, RU) for the header status pill.
func langBadge() string {
	switch state.Current.Language {
	case state.LangTurkish:
		return "TR"
	case state.LangSpanish:
		return "ES"
	case state.LangGerman:
		return "DE"
	case state.LangFrench:
		return "FR"
	case state.LangArabic:
		return "AR"
	case state.LangPortuguese:
		return "PT"
	case state.LangChinese:
		return "ZH"
	case state.LangJapanese:
		return "JA"
	case state.LangItalian:
		return "IT"
	case state.LangRussian:
		return "RU"
	default:
		return "EN"
	}
}
