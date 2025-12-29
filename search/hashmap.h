#ifndef HASH_MAP_H
#define HASH_MAP_H

#include "vector.h"
#include "string.h"
#include <cstddef>

template<typename K, typename V>
class Pair {
public:
    K key;
    V value;

    Pair() {}
    Pair(const K& k, const V& v) : key(k), value(v) {}
};

template<typename K, typename V>
class HashMap {
private:
    Vector<Pair<K, V>> items;

    int find_index(const K& key) const {
        for (size_t i = 0; i < items.get_size(); ++i) {
            if (items[i].key == key) {
                return i;
            }
        }
        return -1;
    }

public:
    HashMap() {}

    V& operator[](const K& key) {
        int idx = find_index(key);
        if (idx != -1) {
            return items[idx].value;
        }
        items.push_back(Pair<K, V>(key, V()));
        return items[items.get_size() - 1].value;
    }

    bool contains(const K& key) const {
        return find_index(key) != -1;
    }

    V* get(const K& key) {
        int idx = find_index(key);
        return idx != -1 ? &items[idx].value : nullptr;
    }

    const V* get(const K& key) const {
        int idx = find_index(key);
        return idx != -1 ? &items[idx].value : nullptr;
    }

    Vector<K> keys() const {
        Vector<K> result;
        for (size_t i = 0; i < items.get_size(); ++i) {
            result.push_back(items[i].key);
        }
        return result;
    }

    size_t size() const {
        return items.get_size();
    }

    bool empty() const {
        return items.empty();
    }

    void clear() {
        items.clear();
    }
};

#endif
