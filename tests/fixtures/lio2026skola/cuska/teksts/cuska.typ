#import "/template/template.typ": *

#let (
    conf,
    print_example, 
    print_example_raw,
    subtask_restriction_table, 
    restrictions_and_requirements,
    contest,
    task,
) = prepare_task_document(
    contest_yaml: "/contest.yaml",
    task_codename: "cuska",
)

#show: doc => conf(
    doc,
)

#v(-1.3em)
#align(right,image("../../template/zvaigznes_4.png",height:1em,))

Mitrā, aukstā purvā dzīvo veģetāra čūska, kurai ļoti garšo dzērvenes.
Čūskai ir zināmas visas dzērveņu atrašanās vietas, un viņa grib apēst tās visas. Iespējams gan, ka ne visas var sasniegt.
Purvā ir arī _akači_ - bīstamas "dubļu" peļķes. Tajos čūska, protams, negrib ierāpot. Ir auksti.

Purva karti varam aprakstīt kā $N times N$ rūtiņu laukumu.
Akaču rūtiņas kartē apzīmētas ar `#`, bet dzērvenes -- ar `*`.
Ir zināma čūskas sākotnējā pozīcija un galvas virziens. Čūskas galva var skatīties vienā no četriem virzieniem: `>` (pa labi), `v` (uz leju), `<` (pa kreisi), `^` (uz augšu).
Visas pārējās jeb tukšās rūtiņas apzīmēsim ar punktu.

Vienā gājienā čūska var pakustēties vienu rūtiņu uz priekšu (virzienā, kurā tobrīd skatās galva) _vai_ pagriezties pa labi / pa kreisi un tad pakustēties vienu rūtiņu uz priekšu. Šīs darbības apzīmēsim attiecīgi ar `F`, `L`, `R`.
Čūskas garums sākumā ir $1$ -- t.i., tikai galva, bet laika gaitā tā var kļūt garāka, apēdot dzērvenes. Apēdot dzērveni, čūskas garums palielinās par vienu rūtiņu.
Čūskai garumā $>=2$ ir arī "aste", kas kustās līdzi.

Apskatīsim piemēru, kurā tiek veikti $13$ gājieni `FFFLLLRFRFRLL` un apēstas visas dzērvenes.

#align(center)[
    #image("cuska.png", width: 80%)
]

Jūsu uzdevums ir noskaidrot, vai čūska var apēst _visas_ dzērvenes pie dotā laukuma! Ja var,
tad jānoskaidro konkrēti gājieni. Šo gājienu skaits nav "jāminimizē", t.i., gājienu skaits jūsu atbildē drīkst arī pārsniegt mazāko iespējamo. Nedrīkst iziet ārā no purva.
Zināms, ka dzērveņu nav pārāk daudz - ne vairāk kā $8$, bet vismaz viena vienmēr ir.

== Ievaddati

Pirmajā rindā dots laukuma izmērs $N (2 <= N <= 5)$.
Tālāk seko $N$ rindas, kas apraksta laukumu ar simboliem `.`, `#`, `*`, `>`,`<`,`^`,`v` kā minēts stāstā.
Laukumā ir tieši viens no simboliem `>`,`<`,`^`,`v` simbols un $1$ līdz $8$ `*` simboli.

== Izvaddati

Ja apēst visas dzērvenes ir iespējams, drīkst izvadīt jebkuru derīgu variantu, kā čūska, veicot ne vairāk kā $10^5$ soļus, var visas ogas apēst.
Šādā gadījumā izvaddatu pirmajai rindai jāsatur atrastā varianta gājienu skaits, bet otrajai rindai jāsatur pašus gājienus -- simbolus `F`,`L`,`R`.

Citādi, ja apēst visas dzērvenes nav iespējams, tad jāizvada viens vārds "NEVAR".

== Ierobežojumi un prasības

#restrictions_and_requirements()

#pagebreak()

== Piemēri

#grid(
    columns: (70%, 27%),
    gutter: 1em,
    [
        #print_example(
            comment: "Atbilst piemēram uzdevuma tekstā.",
            "cuska.i00a",
        )
    ],
    [
         #print_example(
            input_width: 40%,
            "cuska.i00b",
        )
    ]
)

//#pagebreak()

// == 1. apakšuzdevuma testu ievaddati

// #grid(columns: 2, gutter: 1em, 
//     [
//          #print_example(
//              "x_x.i01a", 
//              output: false,
//          )
//     ],
//     [
//          #print_example(
//              "x_x.i01b", 
//              output: false,
//          )
//     ],
// )
    
// #print_example(
//     "x_x.i01c", 
//     output: false,
// )
    


== Apakšuzdevumi un to vērtēšana

#subtask_restriction_table((
    none,
//   [ Uzdevuma tekstā dotie trīs testi ],
  [ $N=2$ ],
  [ $N=3$ ],
  [ Uz laukuma ir tieši $1$ dzērvene ],
  [ $N=4$ ],
  [ Bez papildu ierobežojumiem ]
))