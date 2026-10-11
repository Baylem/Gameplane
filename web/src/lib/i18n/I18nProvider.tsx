import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { IntlProvider, ReactIntlErrorCode } from "react-intl";
import type { IntlConfig } from "react-intl";
import { I18nProvider as AriaI18nProvider } from "@react-aria/i18n";

import { directionFor, listedLanguages, loadMessages } from "./languages";
import type { CompiledCatalog } from "./languages";
import { resolveLanguage } from "./resolve";
import { readLanguageStorage, writeLanguageStorage } from "./storage";

export interface I18nContextValue {
  language: string;
  setLanguage: (tag: string | null) => void;
  direction: "ltr" | "rtl";
}

// Outside a provider (tests, isolated renders) the dashboard stays English.
const FALLBACK: I18nContextValue = {
  language: "en",
  setLanguage: () => {},
  direction: "ltr",
};

const I18nContext = createContext<I18nContextValue | null>(null);

export function useLanguage(): I18nContextValue {
  return useContext(I18nContext) ?? FALLBACK;
}

// onIntlError keeps the console quiet: missing translations are expected (react-intl
// falls back to defaultMessage), and other formatting problems are only worth a
// warning while developing.
// IntlErrorArg is the union react-intl passes to onError (the IntlError subclasses).
type IntlErrorArg = Parameters<NonNullable<IntlConfig["onError"]>>[0];

export function onIntlError(err: IntlErrorArg): void {
  if (err.code === ReactIntlErrorCode.MISSING_TRANSLATION) return;
  if (import.meta.env.DEV) console.warn(err);
}

export interface I18nProviderProps {
  children: ReactNode;
  userLanguage?: string | null;
  instanceDefault?: string | null;
}

interface LoadedMessages {
  tag: string;
  messages: CompiledCatalog;
}

export function I18nProvider({ children, userLanguage, instanceDefault }: I18nProviderProps) {
  const [choice, setChoice] = useState<string | null>(() => readLanguageStorage().choice);
  const listed = useMemo(() => listedLanguages().map((row) => row.tag), []);

  const language = resolveLanguage({
    userChoice: userLanguage,
    browserChoice: choice,
    navigatorLanguages: navigator.languages,
    instanceDefault,
    listed,
  });
  const direction = directionFor(language);

  const [loaded, setLoaded] = useState<LoadedMessages>({ tag: "en", messages: {} });

  useEffect(() => {
    let cancelled = false;
    void loadMessages(language)
      .then((messages) => {
        if (!cancelled) setLoaded({ tag: language, messages: messages ?? {} });
      })
      .catch(() => {
        // A failed chunk leaves the English defaultMessage text in place.
        if (!cancelled) setLoaded({ tag: language, messages: {} });
      });
    return () => {
      cancelled = true;
    };
  }, [language]);

  useEffect(() => {
    document.documentElement.lang = language;
    document.documentElement.dir = direction;
    writeLanguageStorage({ applied: language });
  }, [language, direction]);

  const setLanguage = useCallback((tag: string | null) => {
    writeLanguageStorage({ choice: tag });
    setChoice(tag);
  }, []);

  const value = useMemo<I18nContextValue>(
    () => ({ language, setLanguage, direction }),
    [language, setLanguage, direction],
  );

  // Children are never remounted on a language change: while the new catalog
  // loads, the previous messages stay in place.
  return (
    <I18nContext.Provider value={value}>
      <IntlProvider locale={language} defaultLocale="en" messages={loaded.messages} onError={onIntlError}>
        <AriaI18nProvider locale={language}>{children}</AriaI18nProvider>
      </IntlProvider>
    </I18nContext.Provider>
  );
}
