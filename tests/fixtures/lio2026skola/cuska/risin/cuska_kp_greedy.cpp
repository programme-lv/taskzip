#include <bits/stdc++.h>
using namespace std;

struct Cell {
    int x;
    int y;
};

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    int n;
    cin>>n;
    vector<string> board(n);
    for (int i = 0; i < n; i++) cin >> board[i];

    const int dx[4] = {1, 0, -1, 0};
    const int dy[4] = {0, 1, 0, -1};

    deque<Cell> snake;
    int dir = -1;
    for (int y = 0; y < n; y++) {
        for (int x = 0; x < n; x++) {
            char c = board[y][x];
            if (c == '>' || c == 'v' || c == '<' || c == '^') {
                if (c == '>') dir = 0;
                else if (c == 'v') dir = 1;
                else if (c == '<') dir = 2;
                else dir = 3;
                snake.push_back({x, y});
                board[y][x] = '.';
            }
        }
    }

    if (dir == -1 || snake.empty()) {
        cout << "NEVAR\n";
        return 0;
    }

    vector<vector<bool>> occupied(n, vector<bool>(n, false));
    occupied[snake.front().y][snake.front().x] = true;

    int cranberries = 0;
    for (int y = 0; y < n; y++)
        for (int x = 0; x < n; x++)
            cranberries += (board[y][x] == '*');

    auto nearest_apple = [&](int x, int y) -> int {
        const int INF = 1e9;
        int best = INF;
        for (int yy = 0; yy < n; yy++) {
            for (int xx = 0; xx < n; xx++) {
                if (board[yy][xx] == '*')
                    best = min(best, abs(xx - x) + abs(yy - y));
            }
        }
        return best;
    };

    auto cell_is_blocked = [&](int x, int y, bool tail_stays) -> bool {
        if (x < 0 || x >= n || y < 0 || y >= n) return true;
        if (board[y][x] == '#') return true;
        if (!occupied[y][x]) return false;
        if (!tail_stays) {
            auto tail = snake.back();
            if (tail.x == x && tail.y == y) return false;
        }
        return true;
    };

    auto free_neighbors_after_move = [&](int hx, int hy, bool tail_stays) -> int {
        int cnt = 0;
        for (int nd = 0; nd < 4; nd++) {
            int nx = hx + dx[nd];
            int ny = hy + dy[nd];
            if (!cell_is_blocked(nx, ny, tail_stays))
                cnt++;
        }
        return cnt;
    };

    string answer;
    const int STEP_LIMIT = 100000;

    while (cranberries > 0) {
        struct Option {
            long long score;
            char move_char;
            int ndir;
            int nx;
            int ny;
            bool apple;
        };
        vector<Option> options;

        const int move_order[3] = {-1, 0, 1}; // Left, Forward, Right (relative)
        const char move_char[3] = {'L', 'F', 'R'};

        for (int i = 0; i < 3; i++) {
            int turn = move_order[i];
            int ndir = (dir + turn + 4) % 4;
            int nx = snake.front().x + dx[ndir];
            int ny = snake.front().y + dy[ndir];
            bool apple_here = (nx >= 0 && nx < n && ny >= 0 && ny < n && board[ny][nx] == '*');
            bool tail_stays = apple_here;
            if (cell_is_blocked(nx, ny, tail_stays)) continue;

            int dist = nearest_apple(nx, ny);
            if (dist >= 1e9) dist = 1000;
            int free_neigh = free_neighbors_after_move(nx, ny, tail_stays);

            long long score = 0;
            score += dist * 100;
            score -= free_neigh * 5;
            if (apple_here) score -= 500;
            score += i; // prefer earlier options on tie

            options.push_back({score, move_char[i], ndir, nx, ny, apple_here});
        }

        if (options.empty()) {
            cout << "NEVAR\n";
            return 0;
        }

        auto best = *min_element(options.begin(), options.end(),
                                 [](const Option& a, const Option& b) {
                                     if (a.score != b.score) return a.score < b.score;
                                     return a.move_char < b.move_char;
                                 });

        dir = best.ndir;
        snake.push_front({best.nx, best.ny});
        occupied[best.ny][best.nx] = true;

        if (best.apple) {
            board[best.ny][best.nx] = '.';
            cranberries--;
        } else {
            auto tail = snake.back();
            occupied[tail.y][tail.x] = false;
            snake.pop_back();
        }

        answer.push_back(best.move_char);
        if ((int)answer.size() > STEP_LIMIT) {
            cout << "NEVAR\n";
            return 0;
        }
    }

    cout << answer.size() << "\n" << answer << "\n";
    return 0;
}

