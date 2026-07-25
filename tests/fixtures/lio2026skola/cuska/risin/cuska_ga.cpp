#include <bits/stdc++.h>

using namespace std;

typedef long long ll;
typedef unsigned long long ull;
typedef pair<int, int> pii;

#define fst first
#define snd second

struct Direction
{
    char symbol;
    int dx, dy;
    char new_symbol;
};

set<string> checked;
int recursion_depth = 0;
int max_depth = 0;

void log_grid(const vector<vector<char>> &grid,
              int head_x = -1, int head_y = -1, int tail_x = -1, int tail_y = -1,
              bool enabled = false)
{
    if (!enabled)
        return;

    cerr << endl;
    for (int i = 1; i < grid.size() - 1; i++)
    {
        for (int j = 1; j < grid.size() - 1; j++)
        {
            cerr << grid[i][j];
        }
        cerr << endl;
    }
    if (head_x != -1)
    {
        cerr << "H " << head_x << " " << head_y << endl
             << "T " << tail_x << " " << tail_y << endl;
    }
}

string path(vector<vector<char>> &grid, int head_x, int head_y, int tail_x, int tail_y, int cranberies_left)
{
    recursion_depth++;
    if (recursion_depth > max_depth) {
        max_depth = recursion_depth;
        if (max_depth % 1000 == 0) {
            // cerr << "Max recursion depth: " << max_depth << endl;
        }
    }
    
    // Prevent stack overflow - limit recursion depth
    if (recursion_depth > 15000) {
        recursion_depth--;
        return "IMPOSSIBLE";
    }
    
    if (!cranberies_left)
    {
        log_grid(grid, head_x, head_y, tail_x, tail_y);
        recursion_depth--;
        return "";
    }

    // Add memory check
    if (checked.size() % 100000 == 0) {
        // cerr << "Checked states: " << checked.size() << endl;
    }
    
    if (checked.size() > 50000000) {
        // cerr << "MEMORY LIMIT: Too many states checked!" << endl;
        return "IMPOSSIBLE";
    }

    // Check whether this state has been processed before
    string state = to_string(head_x) + "," + to_string(head_y) + "," + to_string(tail_x) + "," + to_string(tail_y) + ",";
    for (int i = 1; i < grid.size() - 1; i++)
    {
        for (int j = 1; j < grid.size() - 1; j++)
        {
            state += grid[i][j];
        }
    }
    if (checked.count(state))
    {
        return "IMPOSSIBLE";
    }
    
    try {
        checked.insert(state);
    } catch (const std::bad_alloc& e) {
        // cerr << "MEMORY ERROR: " << e.what() << endl;
        // cerr << "Checked size: " << checked.size() << endl;
        return "IMPOSSIBLE";
    }

    // Try all three moves: left, forward, right
    const struct
    {
        char move;
        Direction dirs[4]; // indexed by current direction (^=0, v=1, <=2, >=3)
    } moves[] = {
        {'L', {{'^', -1, 0, '<'}, {'v', 1, 0, '>'}, {'<', 0, 1, 'v'}, {'>', 0, -1, '^'}}},
        {'F', {{'^', 0, -1, '^'}, {'v', 0, 1, 'v'}, {'<', -1, 0, '<'}, {'>', 1, 0, '>'}}},
        {'R', {{'>', 1, 0, '>'}, {'<', -1, 0, '<'}, {'^', 0, -1, '^'}, {'v', 0, 1, 'v'}}}};

    char current = grid[head_y][head_x];
    int dir_idx = (current == '^') ? 0 : (current == 'v') ? 1
                                     : (current == '<')   ? 2
                                                          : 3;

    for (const auto &move : moves)
    {
        const Direction &dir = move.dirs[dir_idx];

        int new_x = head_x + dir.dx;
        int new_y = head_y + dir.dy;
        char target = grid[new_y][new_x];

        if (target == '.' || target == '*' ||
            new_x == tail_x && new_y == tail_y)
        {
            int old_tail_x = tail_x, old_tail_y = tail_y;
            char old_tail = grid[tail_y][tail_x];

            // Find new tail position (only if not eating cranbery)
            if (target != '*')
            {
                if (grid[tail_y - 1][tail_x] == '^')
                    tail_y--;
                else if (grid[tail_y + 1][tail_x] == 'v')
                    tail_y++;
                else if (grid[tail_y][tail_x - 1] == '<')
                    tail_x--;
                else if (grid[tail_y][tail_x + 1] == '>')
                    tail_x++;
                else
                    // If snake has length 1, tail is at head
                    tail_x = new_x, tail_y = new_y;

                grid[old_tail_y][old_tail_x] = '.';
            }

            // Set new head position
            grid[new_y][new_x] = dir.new_symbol;

            int new_cranberies = (target == '*') ? cranberies_left - 1 : cranberies_left;
            string res = path(grid, new_x, new_y, tail_x, tail_y, new_cranberies);

            // Restore grid state
            grid[new_y][new_x] = target;
            if (target != '*')
            {
                grid[old_tail_y][old_tail_x] = old_tail;
                tail_x = old_tail_x;
                tail_y = old_tail_y;
            }

            if (res != "IMPOSSIBLE")
            {
                log_grid(grid, head_x, head_y, tail_x, tail_y);

                recursion_depth--;
                return move.move + res;
            }
        }
    }

    // log_grid(grid, head_x, head_y, tail_x, tail_y);

    recursion_depth--;
    return "IMPOSSIBLE";
}

void solve()
{
    int n;
    cin >> n;
    
    // cerr << "Grid size: " << n << "x" << n << endl;
    
    vector<vector<char>> grid(n + 2, vector<char>(n + 2, '#'));

    for (int i = 1; i <= n; i++)
    {
        for (int j = 1; j <= n; j++)
        {
            cin >> grid[i][j];
        }
    }

    int head_x, head_y, tail_x, tail_y;
    int cranberies_left = 0;
    for (int i = 1; i <= n; i++)
    {
        for (int j = 1; j <= n; j++)
        {
            if (grid[i][j] == '^' || grid[i][j] == 'v' || grid[i][j] == '<' || grid[i][j] == '>')
            {
                head_x = j;
                head_y = i;
                tail_x = j;
                tail_y = i;
            }

            else if (grid[i][j] == '*')
            {
                cranberies_left++;
            }
        }
    }

    string result = path(grid, head_x, head_y, tail_x, tail_y, cranberies_left);
    
    // cerr << "Final checked states: " << checked.size() << endl;
    
    if (result == "IMPOSSIBLE")
    {
        cout << "NEVAR" << endl;
    }
    else
    {
        cout << result.length() << endl
             << result << endl;
    }

    // cout << checked.size() << endl;
}

int main()
{
    ios_base::sync_with_stdio(false);
    cin.tie(0);
    cout.tie(0);

    // int t;
    // cin >> t;
    // while (t--)
    solve();
}