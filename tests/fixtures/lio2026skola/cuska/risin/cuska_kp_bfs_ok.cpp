#include <iostream>
#include <cassert>
#include <bitset>
#include <queue>
#include <unordered_map>
#include <set>
#include <cstring>
#pragma GCC optimize("O3,unroll-loops")
#pragma GCC target("avx2,bmi,bmi2,popcnt,lzcnt,tune=native")
#define uint32_t unsigned int
#define uint64_t unsigned long long
using namespace std;

const int MAX_N=6;
const int MAX_APPLES=MAX_N*MAX_N/4;

string matrix[MAX_N]; // [y][x]

int apple_bit[MAX_N][MAX_N];
bool apple(int x, int y, uint32_t apple_bm){
    int a_bit = apple_bit[y][x];
    if(a_bit==-1) return false;
    return !((apple_bm>>a_bit)&1);
}

void enum_apples(int n){
    int cnt=0;
    for(int i=0;i<n;i++){
        for(int j=0;j<n;j++){
            if(matrix[i][j]=='*') apple_bit[i][j]=cnt++;
            else apple_bit[i][j]=-1;
        }
    }
}

// we assume that the coordinates in the game
// are in the directions 1) top to bottom, 2) left to right
// the top left corner of grid as in the input is (0,0)
// (x,y). x numbers the columns, y numbers the rows

// encode_pos returns [0;MAX_N*MAX_N*4)
// encodes snake head and direction
// for MAX_N=5, max value is 99
// therefore, we need 7 bits to store it
uint32_t encode_pos(int x, int y, int dir){
    return x*MAX_N*4+y*4+dir;
}

int bits_necessary(int max_value){
    int bits=0;
    while(max_value>0){
        max_value >>= 1;
        bits++;
    }
    return bits;
}

const int POS_BITS=bits_necessary(MAX_N*MAX_N*4-1);
const int H_LEN_BITS=bits_necessary(MAX_APPLES+1); // history length bits
const int A_BM_BITS=MAX_APPLES; // eaten apple bitmask
const int HISTORY_BITS=MAX_APPLES*2;

// encodes the full game state into a single 32-bit integer
uint64_t encode(uint32_t pos, uint32_t apples,
    int h_len, uint32_t hist){

    uint64_t res=hist;
    res=(res<<H_LEN_BITS)|h_len;
    res=(res<<A_BM_BITS)|apples;
    res=(res<<POS_BITS)|pos;
    return res;
}

// decodes the full game state from a single 32-bit integer
void decode(uint64_t code, uint32_t& pos, uint32_t& apples,
    int& h_len, uint32_t& hist){

    pos=code&((1<<POS_BITS)-1);
    code>>=POS_BITS;
    apples=code&((1<<A_BM_BITS)-1);
    code>>=A_BM_BITS;
    h_len=code&((1<<H_LEN_BITS)-1);
    code>>=H_LEN_BITS;
    hist=code;
}

void decode_pos(uint32_t pos, int& x, int& y, int& dir){
    dir=pos%4;
    pos /= 4;
    y=pos%MAX_N;
    x=pos/MAX_N;
}

const int dx[]={1,0,-1,0};
const int dy[]={0,1,0,-1};

void decode_history(int h_len, uint32_t history, int dir_offset[]){
    int dir=0;
    for(int i=0;i<h_len;i++){
        int h_entry=history&3;
        history >>= 2;
        if(h_entry==1) dir=(dir+1)%4;
        if(h_entry==3) dir=(dir+3)%4;
        dir_offset[i]=dir;
    }
}

bool intersects[MAX_APPLES+1][1<<HISTORY_BITS];

void precompute_intersects(){
    bool visited[2*MAX_APPLES+2][2*MAX_APPLES+2];
    for(int j=0;j<(1<<HISTORY_BITS);j++){
        int dir_offset[MAX_APPLES+5];
        decode_history(MAX_APPLES, j, dir_offset);

        memset(visited, false, sizeof(visited));
        visited[1+MAX_APPLES][1+MAX_APPLES]=true;
        int x=0,y=0,dir=0;
        int k=MAX_APPLES+1;
        for(int i=0;i<MAX_APPLES;i++){
            x+=dx[dir], y+=dy[dir];
            dir=dir_offset[i];
            if(visited[x+MAX_APPLES+1][y+MAX_APPLES+1]) {k=i+1; break;}
            visited[x+MAX_APPLES+1][y+MAX_APPLES+1]=true;
        }

        for(int i=0;i<k;i++) intersects[i][j]=false;
        for(int i=k;i<=MAX_APPLES;i++) intersects[i][j]=true;
    }
}

string new_matrix[MAX_N];
void print_state(int n, uint64_t code){
    uint32_t pos, apple_bm;
    int h_len; uint32_t history;
    decode(code, pos, apple_bm, h_len, history);
    int px, py, dir;
    decode_pos(pos, px, py, dir);
    // cout << "pos: " << px << ", " << py << ", dir: " << dir << endl;
    // cout << "apple_bm: " << apple_bm << endl;
    // cout<< "h_len: " << h_len << " " << bitset<12>(history) << endl;
    // cout<<endl;

    for(int i=0;i<n;i++)
        new_matrix[i]=matrix[i];
    new_matrix[py][px]=dir==0?'>':dir==1?'v':dir==2?'<':'^';

    int dir_offset[h_len];
    decode_history(h_len, history, dir_offset);
    int cdir=dir;
    int cx=px, cy=py;
    for(int i=0;i<h_len;i++){
        cx-=dx[cdir], cy-=dy[cdir];
        cdir=(dir_offset[i]+dir)%4;
        new_matrix[cy][cx]=(i+1)+'0';
    }
    for(int i=0;i<n;i++){
        for(int j=0;j<n;j++){
            if(new_matrix[i][j]=='*'&&!apple(j,i,apple_bm))
                new_matrix[i][j]='.';
            cout << new_matrix[i][j];
        }
        cout << endl;
    }
}

int main() {
    int n; cin>>n;
    for(int i=0;i<n;i++)
        cin>>matrix[i];

    precompute_intersects();
    // return 0;
    
    enum_apples(n);
    int apple_cnt=0;
    for(int i=0;i<n;i++){
        for(int j=0;j<n;j++){
            if(apple_bit[i][j]!=-1) apple_cnt++;
        }
    }

    uint32_t pos=0;
    for(int i=0;i<n;i++){
        for(int j=0;j<n;j++){
            int dir=-1;
            if(matrix[i][j]=='>') dir=0;
            if(matrix[i][j]=='v') dir=1;
            if(matrix[i][j]=='<') dir=2;
            if(matrix[i][j]=='^') dir=3;
            if(dir!=-1) {
                pos=encode_pos(j,i,dir);
                matrix[i][j]='.'; // clear the cell
                break;
            }
        }
    }

    unordered_map<uint64_t, pair<uint64_t, int>> vis; // state -> (parent, move)

    // queue<uint64_t> q;
    // q.push(encode(pos, 0, 0, 0));
    vector<uint64_t> q;
    q.push_back(encode(pos, 0, 0, 0));
    vis[encode(pos, 0, 0, 0)]={encode(pos, 0, 0, 0), 0};
    uint64_t finish=0;
    int iterations=0;
    while(!q.empty()){
        iterations++;
        // uint64_t f=q.front(); q.pop();
        uint64_t f=q.back(); q.pop_back();

        uint32_t pos, apple_bm;
        int h_len; uint32_t history;
        decode(f, pos, apple_bm, h_len, history);

        int px, py, dir;
        decode_pos(pos, px, py, dir);
        
        // turn left, go forward, turn right. these are the 3 options
        int dirs[3] = {(dir+3)%4, dir, (dir+1)%4};
        uint32_t nh_entry=0;
        for(int d: dirs){
            nh_entry++; // 01-left, 10-forward, 11-right

            int nx=px+dx[d], ny=py+dy[d]; // new position
            if(nx<0 || nx>=n || ny<0 || ny>=n) continue;
            if(matrix[ny][nx]=='#') continue;
            uint32_t nhistory=history;
            int nhlen=h_len;
            // we now have to advance forward and see if we intersect
            // this also depends on whether we grow longer (there is an apple)
            nhistory=(nhistory<<2)|nh_entry;
            int napple_bm=apple_bm;

            if(apple(nx, ny, apple_bm)) {
                nhlen++; napple_bm |= (1<<apple_bit[ny][nx]);
            }
            nhistory &= ((1<<nhlen*2)-1);
            if(intersects[nhlen][nhistory]) continue;


            int npos=encode_pos(nx, ny, d);
            uint64_t ncode=encode(npos, napple_bm, nhlen, nhistory);


            if(vis.count(ncode)) continue;
            vis[ncode]={f, nh_entry};

            if(napple_bm==(1<<apple_cnt)-1) {
                finish = ncode;
                goto finish_label;
            }
            // q.push(ncode);
            q.push_back(ncode);
        }
    }
    // cout<<iterations<<endl;
    cout<<"NEVAR"<<endl;
    return 0;

    finish_label:
    vector<char> moves;
    while(true){
        bool last=vis[finish].first==finish;
        int h_entry=vis[finish].second;
        if(h_entry==1) moves.push_back('L');
        else if(h_entry==3) moves.push_back('R');
        else if(h_entry==2) moves.push_back('F');
        else assert(last||false);

        // cout<<"finish: "<<finish<<", iterations: "<<iterations<<endl;
        // cout<<"last move "<<moves.back()<<endl;
        // print_state(n, finish);
        // cout<<"---"<<endl;
        
        if(last) break;
        finish=vis[finish].first;
    }

    cout<<moves.size()<<endl;
    for(int i=moves.size()-1;i>=0;i--) cout<<moves[i];
    cout<<endl;
    
}