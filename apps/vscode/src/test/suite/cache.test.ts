import { expect } from "chai";
import { LRUCache, createCache, cacheRegistry } from "../../services/cache";

describe("LRUCache", () => {
  it("stores and retrieves values", () => {
    const cache = new LRUCache<string>({ maxSize: 3 });
    cache.set("a", "1");
    expect(cache.get("a")).to.equal("1");
  });

  it("evicts least recently used when maxSize exceeded", () => {
    const cache = new LRUCache<string>({ maxSize: 2 });
    cache.set("a", "1");
    cache.set("b", "2");
    // Access "a" to make "b" oldest
    expect(cache.get("a")).to.equal("1");
    cache.set("c", "3");
    expect(cache.get("b")).to.equal(undefined);
    expect(cache.get("a")).to.equal("1");
    expect(cache.get("c")).to.equal("3");
  });

  it("supports number keys via generics", () => {
    const cache = new LRUCache<string, number>({ maxSize: 2 });
    cache.set(1, "one");
    expect(cache.get(1)).to.equal("one");
    cache.set(2, "two");
    cache.set(3, "three");
    expect(cache.get(1)).to.equal(undefined);
  });
});

describe("Cache Registry", () => {
  it("registers and clears caches", () => {
    const cache = createCache<string>({ maxSize: 2 });
    cache.set("x", "y");
    expect(cache.size()).to.equal(1);
    cacheRegistry.clearAll();
    expect(cache.size()).to.equal(0);
  });
});
