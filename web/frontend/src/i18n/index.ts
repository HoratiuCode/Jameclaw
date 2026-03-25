import dayjs from "dayjs"
import "dayjs/locale/en"
import localizedFormat from "dayjs/plugin/localizedFormat"
import relativeTime from "dayjs/plugin/relativeTime"
import i18n from "i18next"
import { initReactI18next } from "react-i18next"

import en from "./locales/en.json"

dayjs.extend(relativeTime)
dayjs.extend(localizedFormat)

i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        translation: en,
      },
    },
    lng: "en",
    fallbackLng: "en",
    supportedLngs: ["en"],
    debug: false,

    interpolation: {
      escapeValue: false,
    },
  })

dayjs.locale("en")

export default i18n
