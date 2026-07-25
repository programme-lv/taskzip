#pragma once

#include <algorithm>
#include <array>
#include <cstdint>
#include <cstring>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <vector>

namespace cuska {

inline constexpr int kSnakeSolverMaxN = 6;
inline constexpr int kSnakeSolverMaxApples = kSnakeSolverMaxN * kSnakeSolverMaxN / 4;

struct SnakeSolverResult {
    bool solvable = false;
    std::vector<char> moves;
};

namespace detail {

constexpr int bits_necessary(int max_value) {
    int bits = 0;
    while (max_value > 0) {
        max_value >>= 1;
        ++bits;
    }
    return bits;
}

inline constexpr int MAX_N = kSnakeSolverMaxN;
inline constexpr int MAX_APPLES = kSnakeSolverMaxApples;
inline constexpr int POS_BITS = bits_necessary(MAX_N * MAX_N * 4 - 1);
inline constexpr int H_LEN_BITS = bits_necessary(MAX_APPLES + 1);
inline constexpr int A_BM_BITS = MAX_APPLES;
inline constexpr int HISTORY_BITS = MAX_APPLES * 2;

inline std::array<std::string, MAX_N> matrix_storage{};
inline int apple_bit[MAX_N][MAX_N]{};

inline constexpr int dx[] = {1, 0, -1, 0};
inline constexpr int dy[] = {0, 1, 0, -1};

inline bool intersects[MAX_APPLES + 1][1 << HISTORY_BITS]{};
inline bool intersects_ready = false;

struct StateInfo {
    uint64_t parent = 0;
    int move = 0;
};

inline uint32_t encode_pos(int x, int y, int dir) {
    return static_cast<uint32_t>(x * MAX_N * 4 + y * 4 + dir);
}

inline void decode_pos(uint32_t pos, int& x, int& y, int& dir) {
    dir = pos % 4;
    pos /= 4;
    y = pos % MAX_N;
    x = pos / MAX_N;
}

inline uint64_t encode_state(uint32_t pos, uint32_t apples, int h_len, uint32_t history) {
    uint64_t res = history;
    res = (res << H_LEN_BITS) | static_cast<uint64_t>(h_len);
    res = (res << A_BM_BITS) | apples;
    res = (res << POS_BITS) | pos;
    return res;
}

inline void decode_state(uint64_t code, uint32_t& pos, uint32_t& apples, int& h_len, uint32_t& history) {
    pos = static_cast<uint32_t>(code & ((1ULL << POS_BITS) - 1));
    code >>= POS_BITS;
    apples = static_cast<uint32_t>(code & ((1ULL << A_BM_BITS) - 1));
    code >>= A_BM_BITS;
    h_len = static_cast<int>(code & ((1ULL << H_LEN_BITS) - 1));
    code >>= H_LEN_BITS;
    history = static_cast<uint32_t>(code);
}

inline void decode_history(int h_len, uint32_t history, int* dir_offset) {
    int dir = 0;
    for (int i = 0; i < h_len; ++i) {
        int h_entry = static_cast<int>(history & 3);
        history >>= 2;
        if (h_entry == 1) {
            dir = (dir + 1) % 4;
        }
        if (h_entry == 3) {
            dir = (dir + 3) % 4;
        }
        dir_offset[i] = dir;
    }
}

inline void enum_apples(int n) {
    int cnt = 0;
    for (int i = 0; i < n; ++i) {
        for (int j = 0; j < n; ++j) {
            if (matrix_storage[i][j] == '*') {
                apple_bit[i][j] = cnt++;
            } else {
                apple_bit[i][j] = -1;
            }
        }
    }
}

inline bool apple(int x, int y, uint32_t apple_bm) {
    int a_bit = apple_bit[y][x];
    if (a_bit == -1) {
        return false;
    }
    return ((apple_bm >> a_bit) & 1U) == 0;
}

inline void ensure_intersects() {
    if (intersects_ready) {
        return;
    }

    bool visited[2 * MAX_APPLES + 2][2 * MAX_APPLES + 2]{};
    std::array<int, MAX_APPLES + 5> dir_offset{};

    for (int history = 0; history < (1 << HISTORY_BITS); ++history) {
        decode_history(MAX_APPLES, static_cast<uint32_t>(history), dir_offset.data());

        std::memset(visited, 0, sizeof(visited));
        visited[1 + MAX_APPLES][1 + MAX_APPLES] = true;
        int x = 0;
        int y = 0;
        int dir = 0;
        int k = MAX_APPLES + 1;
        for (int i = 0; i < MAX_APPLES; ++i) {
            x += dx[dir];
            y += dy[dir];
            dir = dir_offset[i];
            if (visited[x + MAX_APPLES + 1][y + MAX_APPLES + 1]) {
                k = i + 1;
                break;
            }
            visited[x + MAX_APPLES + 1][y + MAX_APPLES + 1] = true;
        }

        for (int i = 0; i < k; ++i) {
            intersects[i][history] = false;
        }
        for (int i = k; i <= MAX_APPLES; ++i) {
            intersects[i][history] = true;
        }
    }

    intersects_ready = true;
}

inline std::vector<char> reconstruct_moves(uint64_t finish, const std::unordered_map<uint64_t, StateInfo>& vis) {
    std::vector<char> moves;
    while (true) {
        auto it = vis.find(finish);
        if (it == vis.end()) {
            break;
        }
        const StateInfo& info = it->second;
        bool at_root = (info.parent == finish);
        if (info.move == 1) {
            moves.push_back('L');
        } else if (info.move == 2) {
            moves.push_back('F');
        } else if (info.move == 3) {
            moves.push_back('R');
        } else if (!at_root) {
            throw std::runtime_error("Invalid move encoding in solver state");
        }

        if (at_root) {
            break;
        }
        finish = info.parent;
    }
    std::reverse(moves.begin(), moves.end());
    return moves;
}

}  // namespace detail

inline SnakeSolverResult find_shortest_solution(const std::vector<std::string>& board) {
    SnakeSolverResult result;
    const int n = static_cast<int>(board.size());
    if (n <= 0 || n > kSnakeSolverMaxN) {
        return result;
    }
    for (const auto& row : board) {
        if (static_cast<int>(row.size()) != n) {
            return result;
        }
    }

    detail::ensure_intersects();

    for (int i = 0; i < detail::MAX_N; ++i) {
        if (i < n) {
            detail::matrix_storage[i] = board[i];
        } else {
            detail::matrix_storage[i].assign(n, '#');
        }
    }

    detail::enum_apples(n);

    int apple_cnt = 0;
    for (int i = 0; i < n; ++i) {
        for (int j = 0; j < n; ++j) {
            if (detail::matrix_storage[i][j] == '*') {
                ++apple_cnt;
            }
        }
    }
    if (apple_cnt > detail::MAX_APPLES) {
        return result;
    }
    if (apple_cnt == 0) {
        result.solvable = true;
        return result;
    }

    uint32_t start_pos = 0;
    bool head_found = false;
    for (int i = 0; i < n && !head_found; ++i) {
        for (int j = 0; j < n; ++j) {
            int dir = -1;
            if (detail::matrix_storage[i][j] == '>') dir = 0;
            if (detail::matrix_storage[i][j] == 'v') dir = 1;
            if (detail::matrix_storage[i][j] == '<') dir = 2;
            if (detail::matrix_storage[i][j] == '^') dir = 3;
            if (dir != -1) {
                start_pos = detail::encode_pos(j, i, dir);
                detail::matrix_storage[i][j] = '.';
                head_found = true;
                break;
            }
        }
    }
    if (!head_found) {
        return result;
    }

    const uint32_t start_state = detail::encode_state(start_pos, 0, 0, 0);
    std::unordered_map<uint64_t, detail::StateInfo> vis;
    vis.reserve(1 << 14);
    vis[start_state] = {start_state, 0};

    std::vector<uint64_t> queue;
    queue.push_back(start_state);
    size_t head = 0;

    const int full_apples_mask = (apple_cnt == 0) ? 0 : ((1 << apple_cnt) - 1);
    uint64_t finish = 0;
    bool found = false;

    while (head < queue.size() && !found) {
        uint64_t current = queue[head++];
        uint32_t pos, apple_bm;
        int h_len;
        uint32_t history;
        detail::decode_state(current, pos, apple_bm, h_len, history);

        int px, py, dir;
        detail::decode_pos(pos, px, py, dir);
        int dirs[3] = {(dir + 3) % 4, dir, (dir + 1) % 4};
        int move_code = 0;
        for (int ndir : dirs) {
            ++move_code;
            int nx = px + detail::dx[ndir];
            int ny = py + detail::dy[ndir];
            if (nx < 0 || nx >= n || ny < 0 || ny >= n) {
                continue;
            }
            if (detail::matrix_storage[ny][nx] == '#') {
                continue;
            }

            uint32_t nhistory = history;
            int nhlen = h_len;
            nhistory = (nhistory << 2) | static_cast<uint32_t>(move_code);
            uint32_t napple_bm = apple_bm;
            if (detail::apple(nx, ny, apple_bm)) {
                ++nhlen;
                napple_bm |= (1U << detail::apple_bit[ny][nx]);
            }

            int history_bits = nhlen * 2;
            if (history_bits == 0) {
                nhistory = 0;
            } else {
                nhistory &= static_cast<uint32_t>((1U << history_bits) - 1U);
            }
            if (detail::intersects[nhlen][nhistory]) {
                continue;
            }

            uint32_t npos = detail::encode_pos(nx, ny, ndir);
            uint64_t ncode = detail::encode_state(npos, napple_bm, nhlen, nhistory);
            if (vis.count(ncode)) {
                continue;
            }
            vis[ncode] = {current, move_code};

            if (napple_bm == static_cast<uint32_t>(full_apples_mask)) {
                finish = ncode;
                found = true;
                break;
            }

            queue.push_back(ncode);
        }
    }

    if (!found) {
        return result;
    }

    result.moves = detail::reconstruct_moves(finish, vis);
    result.solvable = true;
    return result;
}

inline int shortest_path_length(const std::vector<std::string>& board) {
    auto res = find_shortest_solution(board);
    return res.solvable ? static_cast<int>(res.moves.size()) : -1;
}

}  // namespace cuska


