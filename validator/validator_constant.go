package validator

type CountryCode string

const (
	CCAR   CountryCode = "ar"
	CCEN   CountryCode = "en"
	CCES   CountryCode = "es"
	CCFA   CountryCode = "fa"
	CCFR   CountryCode = "fr"
	CCID   CountryCode = "id"
	CCIT   CountryCode = "it"
	CCJA   CountryCode = "ja"
	CCLV   CountryCode = "lv"
	CCNL   CountryCode = "nl"
	CCPT   CountryCode = "pt"
	CCPTBR CountryCode = "pt_BR"
	CCRU   CountryCode = "ru"
	CCTR   CountryCode = "tr"
	CCVI   CountryCode = "vi"
	CCZH   CountryCode = "zh"
	CCZHTW CountryCode = "zh_TW"
)

func (cc CountryCode) String() string {
	return string(cc)
}
