import { createHighlighter } from 'shiki';

export type SharedCodeBlockLanguage =
  'shell' | 'yaml' | 'json' | 'plaintext' | 'dockerfile' | 'diff' | 'ini' | 'markdown' | (string & {});

type SharedCodeBlockTheme = 'github-dark' | 'github-light';

let highlighterPromise: Promise<Awaited<ReturnType<typeof createHighlighter>>> | null = null;

function resolveShikiTheme(themeMode: 'dark' | 'light') {
  return themeMode === 'dark' ? 'github-dark' : 'github-light';
}

function normalizeLanguage(lang: SharedCodeBlockLanguage) {
  switch (lang) {
    case 'shell':
      return 'shellscript';
    case 'plaintext':
      return null;
    default:
      return lang;
  }
}

async function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      langs: ['diff', 'dockerfile', 'ini', 'json', 'markdown', 'shellscript', 'yaml'],
      themes: ['github-dark', 'github-light'],
    });
  }

  return highlighterPromise;
}

export async function renderHighlightedCodeBlock(options: {
  code: string;
  lang: SharedCodeBlockLanguage;
  themeMode: 'dark' | 'light';
}) {
  const lang = normalizeLanguage(options.lang);
  if (!lang) {
    return '';
  }

  const highlighter = await getHighlighter();
  return highlighter.codeToHtml(options.code, {
    lang,
    theme: resolveShikiTheme(options.themeMode) as SharedCodeBlockTheme,
  });
}
