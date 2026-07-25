#include "testlib.h"
#include <algorithm>
#include <fstream>
#include <iostream>
#include <string>
#include <utility>
#include <vector>

#include "solver.hpp"

using namespace std;

using Cell = pair<int, int>;

vector<string> load_board(const string& path, int target_id, int N) {
    ifstream in(path);
    ensuref(in.good(), "Failed to open board file %s", path.c_str());

    string id_token;
    while (in >> id_token) {
        vector<string> board(N);
        for (int row = 0; row < N; ++row) {
            string row_data;
            bool ok = static_cast<bool>(in >> row_data);
            ensuref(
                ok,
                "Unexpected EOF while reading board %s from %s",
                id_token.c_str(),
                path.c_str()
            );
            board[row] = row_data;
            ensuref(
                static_cast<int>(board[row].size()) == N,
                "Row %d in board %s has wrong length (expected %d, got %d)",
                row,
                id_token.c_str(),
                N,
                static_cast<int>(board[row].size())
            );
        }

        int current_id = stoi(id_token);
        if (current_id == target_id) {
            return board;
        }
    }

    ensuref(false, "Board %d not found in %s", target_id, path.c_str());
    return {};
}

vector<Cell> collect_free_cells(const vector<string>& board) {
    vector<Cell> cells;
    for (int r = 0; r < static_cast<int>(board.size()); ++r) {
        for (int c = 0; c < static_cast<int>(board[r].size()); ++c) {
            if (board[r][c] == '.') {
                cells.emplace_back(r, c);
            }
        }
    }
    return cells;
}

int main(int argc, char* argv[]) {
    registerGen(argc, argv, 1);
    ensuref(argc >= 5, "Usage: %s boards_dir N board_id min_cranberries max_cranberries [tg]", argv[0]);

    const int N = atoi(argv[1]);        // board size
    const int board_id = atoi(argv[2]); // board number within the file
    const int min_cranberries = atoi(argv[3]);
    int max_cranberries = atoi(argv[4]);
    const string boards_dir = ".";
    const string board_file = boards_dir + "/N" + to_string(N) + ".txt";

    vector<string> base_board = load_board(board_file, board_id, N);
    vector<Cell> base_free = collect_free_cells(base_board);
    ensuref(!base_free.empty(), "Board %d has no empty cells for head placement", board_id);

    const int max_placeable = static_cast<int>(base_free.size()) - 1;
    const int solver_limit = min(max_placeable, cuska::kSnakeSolverMaxApples);
    ensuref(
        solver_limit >= min_cranberries,
        "Board %d must allow at least %d cranberries within solver constraints",
        board_id,
        min_cranberries
    );

    max_cranberries = min(max_cranberries, solver_limit);
    ensuref(
        max_cranberries >= min_cranberries,
        "Board %d must have room for at least %d cranberries in addition to the head",
        board_id,
        min_cranberries
    );

    const vector<char> headings = {'^', 'v', '<', '>'};
    const int attempt_count = 100;
    int best_length = -1;
    vector<string> best_board;
    vector<string> last_board = base_board;

    for (int attempt = 0; attempt < attempt_count; ++attempt) {
        vector<string> current_board = base_board;
        vector<Cell> current_free = base_free;

        int head_idx = rnd.next(0, static_cast<int>(current_free.size()) - 1);
        Cell head_cell = current_free[head_idx];
        current_free.erase(current_free.begin() + head_idx);
        current_board[head_cell.first][head_cell.second] =
            headings[rnd.next(0, static_cast<int>(headings.size()) - 1)];

        int attempt_max = min(max_cranberries, static_cast<int>(current_free.size()));
        int cranberry_count = rnd.next(min_cranberries, attempt_max);
        vector<int> order = current_free.empty() ? vector<int>() : rnd.perm(static_cast<int>(current_free.size()));

        for (int i = 0; i < cranberry_count; ++i) {
            Cell cell = current_free[order[i]];
            current_board[cell.first][cell.second] = '*';
        }

        last_board = current_board;
        int path_len = cuska::shortest_path_length(current_board);
        if (path_len < 0) {
            continue;
        }
        if (path_len > best_length) {
            best_length = path_len;
            best_board = current_board;
        }
    }

    cout << N << '\n';
    const vector<string>& output_board = (best_length >= 0) ? best_board : last_board;
    for (const auto& row : output_board) {
        cout << row << '\n';
    }

    return 0;
}