#include "testlib.h"
#include <deque>
#include <sstream>
#include <string>
#include <vector>

using namespace std;

namespace {

constexpr int kMaxSteps = 100000;

enum class OutputSource { Contestant, Jury };

struct Answer {
    bool possible = false;
    int steps = 0;
    string moves;
};

Answer readAnswer(InStream& stream, OutputSource source) {
    Answer ans;
    const bool contest_stream = (source == OutputSource::Contestant);
    if (stream.seekEof()) {
        contest_stream ? stream.quitf(_pe, "Empty output") : stream.quitf(_fail, "Empty output");
    }
    string first = stream.readToken("[^\\s]+", "answer");
    if (first == "NEVAR") {
        ans.possible = false;
        stream.skipBlanks();
        if (!stream.seekEof()) {
            contest_stream ? stream.quitf(_pe, "Extra data after NEVAR") :
                             stream.quitf(_fail, "Extra data after NEVAR");
        }
        return ans;
    }

    long long steps = 0;
    bool number_ok = false;
    try {
        size_t idx = 0;
        steps = stoll(first, &idx);
        number_ok = (idx == first.size());
    } catch (...) {
        number_ok = false;
    }
    if (!number_ok || steps < 0 || steps > kMaxSteps) {
        contest_stream ? stream.quitf(_pe, "Invalid step count: %s", first.c_str()) :
                         stream.quitf(_fail, "Invalid step count: %s", first.c_str());
    }

    if (stream.seekEof()) {
        contest_stream ? stream.quitf(_pe, "Missing move sequence") :
                         stream.quitf(_fail, "Missing move sequence");
    }

    string moves = stream.readToken("[LRF]+", "moves");
    if (static_cast<int>(moves.size()) != steps) {
        contest_stream
            ? stream.quitf(
                  _pe,
                  "Declared %lld steps but provided %zu moves",
                  steps,
                  moves.size()
              )
            : stream.quitf(
                  _fail,
                  "Declared %lld steps but provided %zu moves",
                  steps,
                  moves.size()
              );
    }
    for (char ch : moves) {
        if (ch != 'L' && ch != 'R' && ch != 'F') {
            contest_stream ? stream.quitf(_pe, "Invalid move character: %c", ch) :
                             stream.quitf(_fail, "Invalid move character: %c", ch);
        }
    }
    stream.skipBlanks();
    if (!stream.seekEof()) {
        contest_stream ? stream.quitf(_pe, "Extra data after moves") :
                         stream.quitf(_fail, "Extra data after moves");
    }

    ans.possible = true;
    ans.steps = static_cast<int>(steps);
    ans.moves = moves;
    return ans;
}

struct SimResult {
    bool ok = false;
    string message;
};

SimResult simulate(const vector<string>& board, const string& moves) {
    const int n = static_cast<int>(board.size());
    const int dx[4] = {1, 0, -1, 0};
    const int dy[4] = {0, 1, 0, -1};

    vector<vector<bool>> apple(n, vector<bool>(n, false));
    vector<vector<bool>> occupied(n, vector<bool>(n, false));

    int head_x = -1;
    int head_y = -1;
    int dir = -1;
    int total_apples = 0;

    auto setDir = [](char ch) -> int {
        switch (ch) {
            case '>': return 0;
            case 'v': return 1;
            case '<': return 2;
            case '^': return 3;
            default: return -1;
        }
    };

    for (int y = 0; y < n; ++y) {
        for (int x = 0; x < n; ++x) {
            const char cell = board[y][x];
            if (cell == '*') {
                apple[y][x] = true;
                ++total_apples;
            } else if (cell == '#') {
                // obstacle, nothing to mark
            } else {
                int d = setDir(cell);
                if (d != -1) {
                    head_x = x;
                    head_y = y;
                    dir = d;
                }
            }
        }
    }

    if (head_x == -1 || dir == -1) {
        return {false, "Missing snake head in input"};
    }

    deque<pair<int, int>> snake;
    snake.emplace_front(head_y, head_x);
    occupied[head_y][head_x] = true;
    int eaten = 0;
    int x = head_x;
    int y = head_y;

    for (int i = 0; i < static_cast<int>(moves.size()); ++i) {
        const char mv = moves[i];
        if (mv == 'L') {
            dir = (dir + 3) % 4;
        } else if (mv == 'R') {
            dir = (dir + 1) % 4;
        }
        x += dx[dir];
        y += dy[dir];

        if (x < 0 || x >= n || y < 0 || y >= n) {
            ostringstream oss;
            oss << "Step " << (i + 1) << " leaves the board";
            return {false, oss.str()};
        }
        if (board[y][x] == '#') {
            ostringstream oss;
            oss << "Step " << (i + 1) << " hits a trap";
            return {false, oss.str()};
        }

        const bool grows = apple[y][x];
        if (grows) {
            apple[y][x] = false;
            ++eaten;
        }

        if (!grows) {
            const auto tail = snake.back();
            occupied[tail.first][tail.second] = false;
            snake.pop_back();
        }

        if (occupied[y][x]) {
            ostringstream oss;
            oss << "Step " << (i + 1) << " collides with the snake body";
            return {false, oss.str()};
        }

        snake.emplace_front(y, x);
        occupied[y][x] = true;
    }

    if (eaten != total_apples) {
        ostringstream oss;
        oss << "Only " << eaten << " of " << total_apples << " cranberries eaten";
        return {false, oss.str()};
    }

    return {true, ""};
}

}  // namespace

int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);

    const int n = inf.readInt(2, 5, "n");
    vector<string> board(n);
    int head_count = 0;

    const string rowPattern = "[.#*<>^v]+";
    for (int i = 0; i < n; ++i) {
        board[i] = inf.readToken(rowPattern, "row");
        if (static_cast<int>(board[i].size()) != n) {
            quitf(_fail, "Row %d has invalid length", i + 1);
        }
        for (char ch : board[i]) {
            if (ch == '>' || ch == 'v' || ch == '<' || ch == '^') {
                ++head_count;
            } else if (ch != '.' && ch != '#' && ch != '*') {
                quitf(_fail, "Invalid cell character: %c", ch);
            }
        }
    }
    if (head_count != 1) {
        quitf(_fail, "Input has %d snake heads", head_count);
    }

    const Answer jury = readAnswer(ans, OutputSource::Jury);
    const Answer contestant = readAnswer(ouf, OutputSource::Contestant);

    if (!jury.possible) {
        if (contestant.possible) {
            const SimResult sim = simulate(board, contestant.moves);
            if (sim.ok) {
                quitf(_fail, "Contestant found a solution but jury claims NEVAR");
            }
            quitf(_wa, "Jury says NEVAR and contestant output is invalid: %s", sim.message.c_str());
        }
        quitf(_ok, "Both outputs report NEVAR");
    }

    if (!contestant.possible) {
        quitf(_wa, "Jury has a solution but contestant answered NEVAR");
    }

    const SimResult sim = simulate(board, contestant.moves);
    if (!sim.ok) {
        quitf(_wa, "%s", sim.message.c_str());
    }

    quitf(_ok, "Valid path with %d steps", contestant.steps);
}

