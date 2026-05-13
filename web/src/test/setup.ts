import { afterEach, vi } from 'vitest';

afterEach(() => {
  vi.restoreAllMocks();
});

class ResizeObserverMock {
  disconnect() {}

  observe() {}

  unobserve() {}
}

vi.stubGlobal('ResizeObserver', ResizeObserverMock);
vi.stubGlobal(
  'matchMedia',
  vi.fn().mockImplementation(() => ({
    addEventListener: vi.fn(),
    addListener: vi.fn(),
    dispatchEvent: vi.fn(),
    matches: false,
    media: '',
    onchange: null,
    removeEventListener: vi.fn(),
    removeListener: vi.fn(),
  })),
);
