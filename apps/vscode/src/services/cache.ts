/**
 * LRU (Least Recently Used) Cache implementation
 *
 * Prevents unbounded memory growth by automatically evicting
 * the least recently used items when max size is exceeded.
 *
 * @example
 * ```typescript
 * const cache = new LRUCache<string>({ maxSize: 100 });
 * cache.set('key1', 'value1');
 * const value = cache.get('key1'); // 'value1'
 * cache.clear(); // Remove all items
 * ```
 */

// ============================================================================
// Types & Interfaces
// ============================================================================

export interface CacheOptions<K extends string | number = string> {
  /**
   * Maximum number of items to store in cache.
   * When exceeded, least recently used items are evicted.
   */
  maxSize: number;

  /**
   * Optional callback invoked when an item is evicted.
   * Useful for cleanup logic (closing connections, etc.)
   */
  onEvict?: (key: K, value: any) => void;
}

export interface CacheStats {
  /** Number of successful cache hits */
  hits: number;

  /** Number of cache misses */
  misses: number;

  /** Number of items evicted */
  evictions: number;

  /** Current number of items in cache */
  size: number;

  /** Maximum size of cache */
  maxSize: number;

  /** Hit rate (0-1) */
  hitRate: number;
}

interface CacheEntry<T> {
  value: T;
  timestamp: number;
}

// ============================================================================
// LRU Cache Implementation
// ============================================================================

export class LRUCache<T, K extends string | number = string> {
  private cache: Map<K, CacheEntry<T>>;
  private readonly maxSize: number;
  private readonly onEvict?: (key: K, value: T) => void;
  private accessCounter: number = 0;

  // Statistics
  private stats = {
    hits: 0,
    misses: 0,
    evictions: 0,
  };

  // --------------------------------------------------------------------------
  // Construction
  // --------------------------------------------------------------------------

  constructor(options: CacheOptions<K>) {
    if (options.maxSize <= 0) {
      throw new Error("Cache maxSize must be > 0");
    }

    this.maxSize = options.maxSize;
    this.onEvict = options.onEvict;
    this.cache = new Map();
  }

  // --------------------------------------------------------------------------
  // Public API
  // --------------------------------------------------------------------------

  /**
   * Get a value from the cache.
   * Updates the item's timestamp if found (marks as recently used).
   *
   * @param key - The key to retrieve
   * @returns The cached value, or undefined if not found
   */
  get(key: K): T | undefined {
    const entry = this.cache.get(key);

    if (entry === undefined) {
      this.stats.misses++;
      return undefined;
    }

    // Update timestamp - mark as recently used
    entry.timestamp = ++this.accessCounter;
    this.cache.set(key, entry);

    this.stats.hits++;
    return entry.value;
  }

  /**
   * Set a value in the cache.
   * If cache is full, evicts the least recently used item first.
   *
   * @param key - The key to store
   * @param value - The value to cache
   */
  set(key: K, value: T): void {
    // If updating existing key, just update value and timestamp
    if (this.cache.has(key)) {
      this.cache.set(key, {
        value,
        timestamp: ++this.accessCounter,
      });
      return;
    }

    // If cache is full, evict oldest item
    if (this.cache.size >= this.maxSize) {
      this.evictOldest();
    }

    // Add new entry
    this.cache.set(key, {
      value,
      timestamp: ++this.accessCounter,
    });
  }

  /**
   * Check if a key exists in the cache.
   * Does NOT update timestamp (read-only operation).
   *
   * @param key - The key to check
   * @returns true if key exists, false otherwise
   */
  has(key: K): boolean {
    return this.cache.has(key);
  }

  /**
   * Delete a specific key from the cache.
   *
   * @param key - The key to delete
   * @returns true if key was found and deleted, false otherwise
   */
  delete(key: K): boolean {
    return this.cache.delete(key);
  }

  /**
   * Remove all items from the cache.
   * Optionally invokes onEvict callback for each item.
   *
   * @param invokeCallbacks - Whether to call onEvict for each item (default: false)
   */
  clear(invokeCallbacks: boolean = false): void {
    if (invokeCallbacks && this.onEvict) {
      for (const [key, entry] of this.cache.entries()) {
        this.onEvict(key, entry.value);
      }
    }

    this.cache.clear();
  }

  /**
   * Get the current number of items in the cache.
   *
   * @returns Number of cached items
   */
  size(): number {
    return this.cache.size;
  }

  /**
   * Get cache statistics (hits, misses, evictions, etc.)
   *
   * @returns Cache statistics object
   */
  getStats(): CacheStats {
    const totalRequests = this.stats.hits + this.stats.misses;
    const hitRate = totalRequests > 0 ? this.stats.hits / totalRequests : 0;

    return {
      hits: this.stats.hits,
      misses: this.stats.misses,
      evictions: this.stats.evictions,
      size: this.cache.size,
      maxSize: this.maxSize,
      hitRate,
    };
  }

  /**
   * Reset cache statistics to zero.
   * Does NOT clear cache contents.
   */
  resetStats(): void {
    this.stats = {
      hits: 0,
      misses: 0,
      evictions: 0,
    };
  }

  // --------------------------------------------------------------------------
  // Private Methods
  // --------------------------------------------------------------------------

  /**
   * Evict the least recently used item from the cache.
   * Invokes onEvict callback if provided.
   */
  private evictOldest(): void {
    let oldestKey: K | null = null;
    let oldestTimestamp = Infinity;

    // Find the least recently used item
    for (const [key, entry] of this.cache.entries()) {
      if (entry.timestamp < oldestTimestamp) {
        oldestTimestamp = entry.timestamp;
        oldestKey = key;
      }
    }

    // Evict the oldest item
    if (oldestKey !== null) {
      const entry = this.cache.get(oldestKey);
      this.cache.delete(oldestKey);
      this.stats.evictions++;

      // Call eviction callback if provided
      if (entry && this.onEvict) {
        this.onEvict(oldestKey, entry.value);
      }
    }
  }

  /**
   * Get a snapshot array of all entries as [key, value] tuples.
   */
  entries(): Array<[K, T]> {
    const result: Array<[K, T]> = [];
    for (const [key, entry] of this.cache.entries()) {
      result.push([key, entry.value]);
    }
    return result;
  }

  /**
   * Iterate over all entries, invoking callback with (key, value).
   */
  forEach(callback: (key: K, value: T) => void): void {
    for (const [key, entry] of this.cache.entries()) {
      callback(key, entry.value);
    }
  }
}

// ============================================================================
// Global Cache Registry
// ============================================================================

/**
 * Global registry for tracking all caches in the extension.
 * Used for clearing all caches on workspace changes.
 */
class CacheRegistry {
  private caches = new Set<LRUCache<any, any>>();

  register(cache: LRUCache<any, any>): void {
    this.caches.add(cache);
  }

  unregister(cache: LRUCache<any, any>): void {
    this.caches.delete(cache);
  }

  clearAll(): void {
    for (const cache of this.caches) {
      cache.clear();
    }
  }

  getAll(): LRUCache<any, any>[] {
    return Array.from(this.caches);
  }
}

// Singleton instance
export const cacheRegistry = new CacheRegistry();

/**
 * Helper function to create a registered cache.
 * Cache is automatically added to global registry for cleanup.
 *
 * @param options - Cache configuration options
 * @returns New LRUCache instance
 */
export function createCache<T, K extends string | number = string>(
  options: CacheOptions<K>
): LRUCache<T, K> {
  const cache = new LRUCache<T, K>(options);
  cacheRegistry.register(cache);
  return cache;
}
