let accessToken = '';

export function getAccessToken() {
  if (accessToken) {
    return accessToken;
  }

  try {
    const raw = localStorage.getItem('user');
    if (!raw) {
      return '';
    }

    const persisted = JSON.parse(raw) as { token?: string };
    return persisted.token ?? '';
  } catch {
    return '';
  }
}

export function setAccessToken(token: string) {
  accessToken = token;
}

export function clearAccessToken() {
  accessToken = '';
}
