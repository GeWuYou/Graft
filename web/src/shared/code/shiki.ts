import { createHighlighter } from 'shiki';

export type SharedCodeBlockLanguage =
  'shell' | 'yaml' | 'json' | 'plaintext' | 'dockerfile' | 'diff' | 'ini' | 'markdown' | (string & {});

type SharedCodeBlockTheme = 'github-dark' | 'github-light';

let highlighterPromise: Promise<Awaited<ReturnType<typeof createHighlighter>>> | null = null;

/**
 * 将主题模式映射为对应的 Shiki 主题。
 *
 * @param themeMode - 主题模式
 * @returns 深色模式对应 `github-dark`，浅色模式对应 `github-light`
 */
function resolveShikiTheme(themeMode: 'dark' | 'light'): SharedCodeBlockTheme {
  return themeMode === 'dark' ? 'github-dark' : 'github-light';
}

/**
 * 将代码块语言名称转换为 Shiki 使用的语言标识符。
 *
 * @param lang - 代码块语言名称
 * @returns `shellscript`、`null` 或原始语言名称
 */
function normalizeLanguage(lang: SharedCodeBlockLanguage): SharedCodeBlockLanguage | null {
  switch (lang) {
    case 'shell':
      return 'shellscript';
    case 'plaintext':
      return null;
    default:
      return lang;
  }
}

/**
 * 获取共享的 Shiki 代码高亮器实例。
 *
 * @returns 用于代码高亮的 Shiki 高亮器实例
 */
async function getHighlighter() {
  if (!highlighterPromise) {
    const nextHighlighterPromise = createHighlighter({
      langs: ['diff', 'dockerfile', 'ini', 'json', 'markdown', 'shellscript', 'yaml'],
      themes: ['github-dark', 'github-light'],
    });
    const handledHighlighterPromise = nextHighlighterPromise.catch((error: unknown) => {
      if (highlighterPromise === handledHighlighterPromise) {
        highlighterPromise = null;
      }
      throw error;
    });
    highlighterPromise = handledHighlighterPromise;
  }

  return highlighterPromise;
}

/**
 * 将代码渲染为带语法高亮的 HTML。
 *
 * @param options - 包含代码内容、语言标识和主题模式的渲染选项
 * @returns 语法高亮后的 HTML；语言为 `plaintext` 时返回空字符串
 */
export async function renderHighlightedCodeBlock(options: {
  code: string;
  lang: SharedCodeBlockLanguage;
  themeMode: 'dark' | 'light';
}): Promise<string> {
  const lang = normalizeLanguage(options.lang);
  if (!lang) {
    return '';
  }

  const highlighter = await getHighlighter();
  return highlighter.codeToHtml(options.code, {
    lang,
    theme: resolveShikiTheme(options.themeMode),
  });
}
