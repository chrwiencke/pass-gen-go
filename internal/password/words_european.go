package password

// These lists use lowercase ASCII forms. Accents are intentionally removed so
// generated passphrases remain accepted by password fields that reject Unicode.

var czechWords = []string{
	"auto", "balik", "barva", "bota", "brana", "breh", "budova", "caj",
	"cesta", "chata", "chleb", "chvilka", "clen", "clovek", "darek", "datum",
	"denik", "divadlo", "dopis", "duben", "dum", "dvere", "email", "farma",
	"film", "firma", "forma", "fotka", "hlas", "hodina", "hora", "hotel",
	"hrad", "hrana", "hvezda", "jablko", "jaro", "jazyk", "jedlo", "kabel",
	"kamen", "karta", "kava", "klice", "kniha", "kolo", "kontrola", "kosa",
	"kostel", "kraj", "krok", "kruh", "kurz", "les", "leto", "list",
	"lod", "luka", "mesto", "mince", "mluvit", "most", "mysl", "napad",
	"noc", "obchod", "obraz", "okno", "ovoce", "papir", "park", "pecivo",
	"penize", "plan", "pole", "postel", "prace", "pramen", "pritel", "radost",
	"reka", "rodina", "rok", "ryba", "sad", "skola", "slovo", "strom",
	"stul", "svetlo", "vlak", "voda", "zahrada", "zima",
}

var danishWords = []string{
	"aften", "anker", "arbejde", "avis", "bane", "bank", "barn", "billede",
	"billet", "bjerg", "blad", "blomst", "bog", "bord", "bro", "butik",
	"by", "cykel", "dag", "daler", "dame", "dato", "digt", "drage",
	"dreng", "drift", "dyr", "efter", "eg", "engel", "farve", "felt",
	"ferie", "fest", "film", "fisk", "fjeld", "flaske", "flod", "fly",
	"form", "fornuft", "frakke", "fred", "frisk", "frugt", "fugl", "gave",
	"gade", "glas", "glimt", "gren", "grund", "gulv", "have", "havn",
	"helg", "hest", "himmel", "historie", "hjerte", "hotel", "hus", "hytte",
	"ide", "ild", "jakke", "kaffe", "kage", "kalender", "kamera", "kirke",
	"klasse", "klokke", "knap", "kode", "kontor", "kop", "korn", "kort",
	"kunst", "kyst", "lampe", "land", "leder", "liste", "lys", "marked",
	"morgen", "motor", "navn", "nummer", "papir", "park", "plan", "post",
	"radio", "regn", "rejse", "ring", "skole", "sol", "sten", "vand",
}

var dutchWords = []string{
	"avond", "anker", "appel", "arbeid", "archief", "bakker", "bank", "beeld",
	"berg", "bericht", "beker", "blad", "bloem", "boek", "boom", "boot",
	"bord", "brief", "brug", "buurt", "cijfer", "cirkel", "dag", "dak",
	"datum", "dorp", "draad", "droom", "duin", "eiland", "engel", "fiets",
	"film", "fles", "foto", "fruit", "gang", "glas", "goud", "gras",
	"groep", "haven", "hemel", "herfst", "hoek", "hotel", "huis", "idee",
	"jacht", "jas", "kaart", "kaas", "kamer", "kanaal", "kasteel", "keuken",
	"kist", "klant", "klas", "kleur", "klok", "knoop", "koffie", "koffer",
	"kunst", "kust", "lamp", "land", "leven", "licht", "lijn", "lijst",
	"markt", "melk", "mens", "molen", "morgen", "naam", "nacht", "nummer",
	"papier", "park", "plaats", "poort", "radio", "regen", "reis", "rivier",
	"school", "sleutel", "spoor", "stad", "steen", "straat", "tafel", "water",
}

var finnishWords = []string{
	"aamu", "aika", "akku", "ala", "asema", "avain", "haave", "halli",
	"harju", "hattu", "hiekka", "hissi", "hotelli", "huone", "ikkuna", "ilta",
	"juna", "juttu", "kaakao", "kaari", "kahvi", "kala", "kallio", "kamera",
	"kana", "kartta", "kasvi", "katu", "kauppa", "kello", "kentta", "keski",
	"kesa", "kieli", "kirja", "kivi", "koira", "koti", "kuja", "kukka",
	"kulma", "kurssi", "kuva", "kyla", "laakso", "laiva", "lamppu", "lasi",
	"lehti", "leipa", "linja", "lintu", "loma", "luokka", "maa", "maito",
	"marja", "matka", "meri", "metsa", "mieli", "museo", "nimi", "noro",
	"ovi", "paikka", "pallo", "paperi", "pelto", "pieni", "pilvi", "piste",
	"polku", "puisto", "puu", "radio", "raha", "ranta", "rata", "reitti",
	"ruoka", "sade", "saari", "sana", "silta", "sivu", "talo", "vesi",
}

var frenchWords = []string{
	"abricot", "accord", "adresse", "affaire", "aigle", "album", "ami", "annee",
	"arbre", "argent", "atelier", "avion", "bagage", "banque", "bateau", "beurre",
	"bijou", "billet", "blanc", "bois", "bonjour", "bord", "bouteille", "branche",
	"bureau", "cafe", "cahier", "camion", "carte", "centre", "chaise", "chance",
	"chanson", "chat", "chemin", "chien", "ciel", "citron", "clef", "coeur",
	"colline", "couleur", "cour", "cuisine", "danse", "date", "dessin", "dossier",
	"ecole", "ecran", "eglise", "etoile", "famille", "fenetre", "ferme", "fete",
	"fleur", "forfait", "foret", "fromage", "garage", "gateau", "groupe", "hiver",
	"hotel", "image", "jardin", "journal", "lampe", "lettre", "livre", "lumiere",
	"maison", "marche", "matin", "membre", "mer", "montagne", "moteur", "musique",
	"papier", "parc", "photo", "pierre", "place", "porte", "route", "soleil",
}

var germanWords = []string{
	"abend", "anker", "apfel", "arbeit", "archiv", "auto", "bahn", "bank",
	"baum", "becher", "berg", "bild", "blatt", "blume", "boden", "boot",
	"brief", "bruecke", "buch", "buehne", "dorf", "drache", "ecke", "ei",
	"feld", "ferien", "fest", "film", "flasche", "fluss", "form", "frage",
	"freund", "frost", "garten", "glas", "glueck", "gold", "gras", "gruppe",
	"hafen", "haus", "heft", "herbst", "himmel", "hotel", "idee", "insel",
	"jacke", "jahr", "kaffee", "karte", "keller", "kirche", "klasse", "kleid",
	"koffer", "konto", "korb", "kraft", "kreis", "kunst", "kurs", "lampe",
	"land", "leben", "licht", "linie", "liste", "markt", "mensch", "morgen",
	"motor", "nacht", "name", "nummer", "papier", "park", "platz", "radio",
	"regen", "reise", "ring", "schule", "sonne", "stein", "strasse", "wasser",
}

var hungarianWords = []string{
	"ablak", "alma", "arany", "asztal", "auto", "bank", "barat", "bebor",
	"betu", "bolt", "bor", "borond", "cipo", "csalad", "csillag", "datum",
	"del", "domb", "doboz", "dolog", "ember", "erdei", "erkely", "este",
	"etel", "ev", "falu", "fal", "felho", "feny", "film", "folyo",
	"forma", "foto", "fu", "gomb", "gyerek", "hajo", "hal", "hang",
	"haz", "hegy", "hidas", "hir", "hotel", "ido", "iskola", "jarat",
	"jatek", "jegy", "kert", "kep", "kerek", "kez", "konyv", "kor",
	"kosar", "kristaly", "kutya", "lampa", "lap", "lecke", "level", "lista",
	"lo", "madar", "mez", "motor", "munka", "nev", "nyar", "ablakos",
	"orom", "park", "piac", "posta", "radio", "reggel", "repulo", "sarok",
	"sor", "szel", "sziget", "szoba", "tanar", "ter", "tenger", "viz",
}

var italianWords = []string{
	"acqua", "albero", "amico", "anno", "ape", "arco", "area", "arte",
	"auto", "banca", "barca", "bello", "borsa", "bosco", "bottiglia", "braccio",
	"caffe", "calcio", "camera", "campo", "cane", "carta", "casa", "castello",
	"cielo", "citta", "colore", "conto", "corso", "costa", "data", "dente",
	"disegno", "donna", "dono", "estate", "festa", "fiore", "fiume", "foglia",
	"forma", "forno", "fratello", "frutta", "gatto", "giorno", "gioco", "grano",
	"gruppo", "hotel", "idea", "isola", "lago", "lampada", "latte", "lettera",
	"libro", "lista", "luce", "luna", "madre", "mare", "mercato", "monte",
	"motore", "muro", "notte", "numero", "orto", "pane", "parco", "pasta",
	"pietra", "ponte", "porta", "radio", "ragazzo", "rete", "scuola", "sole",
}

var polishWords = []string{
	"adres", "auto", "bank", "barwa", "bilet", "biuro", "blok", "brama",
	"brzeg", "but", "chleb", "chwila", "cien", "cud", "czas", "data",
	"deszcz", "dom", "droga", "drzewo", "dzien", "ekran", "farba", "film",
	"firma", "forma", "fotka", "gora", "gosc", "grupa", "hotel", "idea",
	"jablko", "jacht", "jezyk", "kabel", "kawa", "kierunek", "klasa", "klucz",
	"kolo", "kolor", "konto", "kosz", "kot", "krok", "ksiazka", "kwiat",
	"las", "lato", "lekcja", "list", "lodka", "luna", "mapa", "miasto",
	"most", "motor", "mysl", "nazwa", "noc", "numer", "obraz", "okno",
	"owoc", "papier", "park", "plan", "plac", "poczta", "pole", "pomysl",
	"praca", "radio", "radosc", "reka", "rodzina", "rok", "ryba", "sklep",
	"szkola", "tabela", "ulica", "woda", "zamek", "zima",
}

var portugueseWords = []string{
	"abrigo", "acordo", "agua", "amigo", "ano", "arvore", "aviao", "banco",
	"barco", "bairro", "bilhete", "bolo", "bolsa", "botao", "caderno", "cafe",
	"caixa", "caminho", "campo", "carta", "casa", "castelo", "cidade", "claro",
	"coisa", "cor", "costa", "data", "dedo", "desenho", "dia", "dinheiro",
	"escola", "espaco", "estrela", "familia", "farol", "festa", "filme", "flor",
	"forma", "foto", "fruta", "garrafa", "grupo", "hotel", "ideia", "ilha",
	"janela", "jardim", "jogo", "jornal", "lago", "lampada", "leite", "letra",
	"livro", "lugar", "luz", "mae", "manha", "mar", "mercado", "mesa",
	"moeda", "monte", "motor", "muro", "noite", "numero", "papel", "parque",
	"pedra", "ponte", "porta", "praia", "radio", "rio", "rua", "sol",
}

var romanianWords = []string{
	"acasa", "apa", "arbore", "arc", "argint", "auto", "banca", "barca",
	"bilet", "bloc", "brat", "cadou", "cafea", "camera", "camp", "carte",
	"casa", "castel", "ceas", "cer", "cheie", "culoare", "cuvant", "data",
	"deal", "desen", "dimineata", "drum", "ecran", "familie", "fereastra", "film",
	"floare", "forma", "frate", "fruct", "gara", "gradina", "grup", "hotel",
	"idee", "insula", "joc", "lac", "lampa", "lapte", "lectie", "lemn",
	"linie", "lista", "luna", "masa", "masina", "mare", "mijloc", "munte",
	"nume", "numar", "oras", "parc", "piata", "piatra", "pod", "poarta",
	"posta", "prieten", "radio", "rau", "retea", "scoala", "seara", "soare",
	"stea", "strada", "tabel", "timp", "tren", "usa", "vara", "zi",
}

var spanishWords = []string{
	"abeja", "abrazo", "agua", "amigo", "ano", "arbol", "arena", "arte",
	"auto", "banco", "barco", "barrio", "bolsa", "botella", "brazo", "cafe",
	"calle", "campo", "carta", "casa", "castillo", "cielo", "ciudad", "clase",
	"coche", "color", "costa", "cuadro", "cuento", "dato", "dia", "dinero",
	"escuela", "espacio", "estrella", "familia", "fiesta", "flor", "forma", "foto",
	"fruta", "gato", "grupo", "hotel", "idea", "isla", "jardin", "juego",
	"lapiz", "leche", "letra", "libro", "lugar", "luz", "madre", "mar",
	"mercado", "mesa", "moneda", "monte", "motor", "mujer", "noche", "numero",
	"papel", "parque", "piedra", "playa", "plaza", "puente", "puerta", "radio",
	"rio", "ruta", "sol", "tren", "ventana", "viaje",
}

var swedishWords = []string{
	"afton", "ankare", "arbete", "arkiv", "bank", "barn", "berg", "bild",
	"biljett", "blad", "blomma", "bok", "bord", "bro", "butik", "by",
	"cykel", "dag", "damm", "datum", "dikt", "djur", "drake", "drom",
	"efter", "ek", "eld", "fabrik", "farg", "ferie", "fest", "film",
	"fisk", "fjall", "flaska", "flod", "flyg", "form", "fred", "frukt",
	"fagel", "gata", "glas", "glimt", "gren", "grund", "grupp", "guld",
	"hage", "hamn", "helg", "himmel", "historia", "hjarta", "hotell", "hus",
	"hylla", "ide", "jacka", "kaffe", "kaka", "kalender", "kamera", "karta",
	"klocka", "klass", "knapp", "kod", "kontor", "kopp", "korn", "kort",
	"konst", "kust", "lampa", "land", "ledare", "lista", "ljus", "marknad",
	"morgon", "motor", "namn", "nummer", "papper", "park", "plan", "vatten",
}
