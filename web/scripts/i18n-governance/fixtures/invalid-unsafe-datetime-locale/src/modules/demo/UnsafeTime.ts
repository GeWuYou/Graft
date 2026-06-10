export const label = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date());
export const fallback = new Date().toLocaleDateString();
