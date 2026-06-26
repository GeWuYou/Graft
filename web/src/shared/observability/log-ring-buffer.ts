export type LogRingBufferView<T> = Readonly<{
  version: number;
  size: number;
  capacity: number;
  oldestSeq: number | null;
  newestSeq: number | null;
  at: (index: number) => T | undefined;
  seqAt: (index: number) => number | undefined;
  toArray: () => readonly T[];
}>;

export type LogRingBufferAppendResult<T> = Readonly<{
  overwritten: T | undefined;
  overwrittenSeq: number | undefined;
  seq: number;
  version: number;
}>;

type RingSlot<T> = {
  seq: number;
  value: T;
};

function assertPositiveCapacity(capacity: number) {
  if (!Number.isInteger(capacity) || capacity <= 0) {
    throw new RangeError('RingBuffer capacity must be a positive integer');
  }
}

/**
 * 提供固定容量、O(1) 写入的环形缓冲区。
 *
 * `snapshot()` 返回只读 live view 而非复制数组，避免在热路径产生额外分配。
 */
export class LogRingBuffer<T> {
  readonly #slots: Array<RingSlot<T> | undefined>;
  readonly #capacity: number;
  #head = 0;
  #size = 0;
  #nextSeq = 1;
  #version = 0;

  constructor(capacity: number) {
    assertPositiveCapacity(capacity);
    this.#capacity = capacity;
    this.#slots = Array.from({ length: capacity });
  }

  append(value: T): LogRingBufferAppendResult<T> {
    const overwrittenSlot = this.#slots[this.#head];
    const seq = this.#nextSeq;

    this.#slots[this.#head] = {
      seq,
      value,
    };
    this.#head = (this.#head + 1) % this.#capacity;
    this.#nextSeq += 1;
    this.#version += 1;

    if (this.#size < this.#capacity) {
      this.#size += 1;
    }

    return {
      overwritten: overwrittenSlot?.value,
      overwrittenSeq: overwrittenSlot?.seq,
      seq,
      version: this.#version,
    };
  }

  overwrite(value: T): LogRingBufferAppendResult<T> {
    return this.append(value);
  }

  clear() {
    this.#slots.fill(undefined);
    this.#head = 0;
    this.#size = 0;
    this.#version += 1;
  }

  size() {
    return this.#size;
  }

  capacity() {
    return this.#capacity;
  }

  version() {
    return this.#version;
  }

  snapshot(): LogRingBufferView<T> {
    const version = () => this.#version;
    const size = () => this.#size;
    const capacity = () => this.#capacity;
    const oldestSeq = () => (this.#size ? (this.#slotAt(0)?.seq ?? null) : null);
    const newestSeq = () => (this.#size ? (this.#slotAt(this.#size - 1)?.seq ?? null) : null);

    const view = {
      get version() {
        return version();
      },
      get size() {
        return size();
      },
      get capacity() {
        return capacity();
      },
      get oldestSeq() {
        return oldestSeq();
      },
      get newestSeq() {
        return newestSeq();
      },
      at: (index: number) => this.#slotAt(index)?.value,
      seqAt: (index: number) => this.#slotAt(index)?.seq,
      toArray: () => {
        const values: T[] = [];

        for (let index = 0; index < this.#size; index += 1) {
          const slot = this.#slotAt(index);
          if (slot) {
            values.push(slot.value);
          }
        }

        return values;
      },
    };

    return Object.freeze(view);
  }

  #slotAt(index: number) {
    if (!Number.isInteger(index) || index < 0 || index >= this.#size) {
      return undefined;
    }

    const start = this.#size === this.#capacity ? this.#head : 0;
    const slotIndex = (start + index) % this.#capacity;

    return this.#slots[slotIndex];
  }
}
