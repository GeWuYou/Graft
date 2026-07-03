import type { ComposerTranslation } from 'vue-i18n';

type Translate = ComposerTranslation;

function formatImportPreviewValue(
  t: Translate,
  category: 'validationStatusValues' | 'canonicalNameSourceValues',
  value: string,
) {
  const key = `project.import.preview.${category}.${value}`;
  const translated = t(key);
  return translated === key ? value : translated;
}

export function formatImportPreviewValidationStatus(t: Translate, status: string) {
  return formatImportPreviewValue(t, 'validationStatusValues', status);
}

export function formatImportPreviewCanonicalNameSource(t: Translate, source: string) {
  return formatImportPreviewValue(t, 'canonicalNameSourceValues', source);
}
