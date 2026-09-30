package knowledge

import "strings"

// Word lists the patterns draw from. None of this is anyone's actual
// username.

// firstNames lists first names and nicknames from many countries (1253).
var firstNames = strings.Fields(`
james john robert michael william david richard joseph thomas charles
christopher daniel matthew anthony mark donald steven paul andrew joshua
kenneth kevin brian george timothy ronald edward jason jeffrey ryan jacob
gary nicholas eric jonathan stephen larry justin scott brandon benjamin
samuel gregory alexander frank patrick raymond jack dennis jerry tyler aaron
adam nathan henry douglas zachary peter kyle noah ethan jeremy walter
christian keith roger terry austin sean gerald carl harold dylan arthur
lawrence jordan jesse bryan billy bruce gabriel joe logan alan albert wayne
randy vincent mason roy ralph bobby russell bradley philip eugene liam
oliver lucas aiden jayden carter owen luke isaac caleb hunter connor landon
evan colton cameron chase blake cole wyatt levi hudson grayson easton jaxon
asher lincoln miles leo max oscar harry charlie alfie freddie archie theo
toby finley reggie teddy louie rory ollie jamie callum declan kieran jake
josh tom ben dan sam alex chris mike matt nick tony steve rob will ed ted
jim tim andy danny tommy jimmy johnny joey kenny ricky jeff greg brad derek
travis troy shane dustin marcus darren trevor spencer seth grant dean neil
ross craig stuart graham gavin colin martin simon ian nigel barry trent reid
brett dale glenn clay beau jace kai ezra silas jasper felix hugo arlo milo
otis rowan finn ronan elliot emmett everett beckett brody bryce tristan
maddox ryder rhys harvey jenson zach nate gabe drew mary patricia jennifer
linda elizabeth barbara susan jessica sarah karen lisa nancy betty sandra
margaret ashley kimberly emily donna michelle carol amanda melissa deborah
stephanie dorothy rebecca sharon laura cynthia amy kathleen angela shirley
brenda emma anna pamela nicole samantha katherine christine helen debra
rachel carolyn janet maria catherine heather diane olivia julie joyce
victoria ruth virginia lauren kelly christina joan evelyn judith andrea
hannah megan cheryl jacqueline martha madison teresa gloria sara janice ann
kathryn abigail sophia frances jean alice judy isabella julia grace amber
denise danielle marilyn beverly charlotte natalie theresa diana brittany
doris kayla alexis lori marie ava mia amelia harper ella chloe lily zoe leah
aubrey addison layla scarlett aria riley nora hazel violet aurora savannah
audrey brooklyn bella claire skylar lucy paisley everly caroline nova
genesis emilia kennedy maya willow kinsley naomi aaliyah elena sadie ariana
allison gabriella madelyn cora ruby eva serenity autumn adeline hailey
gianna valentina isla eliana quinn nevaeh ivy piper lydia alexa josephine
emery delilah vivian stella clara freya poppy daisy evie rosie phoebe millie
florence matilda elsie imogen esme ellie holly jade katie kate jess becky
liz beth kat jen jenny molly abby gemma hayley tilly nell iris ada edith
mabel maisie lottie connie orla niamh siobhan aoife ciara saoirse juan jose
luis miguel carlos jorge pedro manuel francisco antonio javier diego pablo
mateo santiago sebastian nicolas alejandro fernando ricardo eduardo rafael
andres sergio raul alberto roberto hector ruben emilio joaquin ignacio
gonzalo cristian julio cesar marco tomas felipe gustavo rodrigo leonardo
bruno thiago enzo matias benicio santino valentino maximo lorenzo agustin
martina lucia valeria camila sofia daniela gabriela mariana fernanda paula
natalia carolina alejandra ximena renata regina antonella catalina florencia
agustina julieta luciana abril milagros rocio guadalupe carmen rosa ana
beatriz ines pilar marta cristina raquel silvia monica veronica lorena
adriana juliana leticia larissa bianca joao guilherme vinicius caio davi
heitor bernardo murilo matheus renan diogo tiago goncalo afonso duarte bruna
vitoria yasmin manuela helena luiza thais carla pierre louis jules raphael
mathis clement antoine maxime alexandre julien baptiste quentin romain
florian benoit etienne mathieu olivier francois guillaume remi yann louise
lea manon camille juliette lina margaux eloise anais oceane clemence
mathilde amelie aurelie celine elodie sandrine nathalie sophie margot
capucine lukas jonas leon elias moritz julian niklas jan tobias stefan
matthias andreas markus wolfgang klaus jurgen dieter uwe horst hans karl
fritz otto emil anton oskar lars sven erik bjorn nils magnus axel henrik
anders johan mikael viktor gustav ingrid astrid sigrid linnea elsa maja ebba
saga alva wilma hanna lena mila katharina johanna greta frieda ida lotte
fenna sanne femke daan sem bram luuk thijs stijn joost pieter jeroen bas
maarten wouter koen sander giovanni luca matteo alessandro francesco davide
riccardo simone federico gabriele stefano paolo giuseppe salvatore vincenzo
massimo fabio emanuele tommaso filippo pietro edoardo nicolo giulia chiara
francesca alessia giorgia elisa federica beatrice ginevra noemi arianna gaia
camilla paola roberta simona ivan dmitri alexei sergei andrei mikhail
nikolai pavel vladimir yuri oleg boris igor maxim artem kirill denis roman
egor ilya stanislav bogdan taras oleksandr jakub kacper szymon mateusz
bartosz piotr pawel tomasz krzysztof marek lukasz filip wojciech michal
milan luka marko nikola dusan goran dragan olga natasha tatiana svetlana
irina anastasia ekaterina daria polina ksenia alina yulia oksana viktoria
zofia agnieszka katarzyna magdalena aleksandra zuzana petra tereza lucie
ivana jelena milena marija mohammed muhammad ahmed ali omar hassan hussein
yusuf ibrahim khalid mustafa karim tariq rami samir fadi nabil walid ziad
amir hamza bilal zaid faisal saad mahmoud youssef idris ismail rayan ayman
malik jamal nasser farid salim fatima aisha zainab maryam noor huda amira
salma hana leila rania dina mona reem nadia samira jana lara farah mariam
zahra zeynep elif emre mehmet mert burak can cem kerem baris deniz arda ege
selin ecem buse merve irem esra ayse fatma defne reza dariush kian arash
navid parisa shirin nasrin sina omid babak cyrus raj arjun rahul amit vikram
rohan aditya kiran ravi sanjay suresh ramesh rajesh anil sunil vijay ajay
deepak manoj ashok nikhil varun karan akash aman ankit abhishek siddharth
harsh yash arnav ishaan vivaan aarav reyansh krishna ganesh shiva ram laksh
pranav tanmay rishi kunal gaurav sachin rohit hardik imran asif usman zain
ayaan danish rizwan salman shahid tahir farhan sohail priya ananya divya
pooja neha anjali kavya sneha shreya riya isha nisha meera sita lakshmi
deepika priyanka aishwarya kajal pallavi swati shruti tanvi aditi ishita
diya saanvi myra kiara zara ayesha sana hira mahnoor iqra amna sadia nusrat
tasnim farzana sultana wei jun hao min lei jie tao bo ming hui xin yu yan
mei ling xiao jing hua fang na qing ying yun zhen kenji hiroshi takumi
haruto sota yuto riku ren kaito yamato daiki shota kenta takeshi akira hiro
ryo sho yuki sakura yui aoi mio rin haruka nanami misaki ayaka mai emi miyu
saki nana koharu minato jisoo minjun seojun jiho doyun hajun siwoo jiwoo
seoyeon jiwon minseo seoyun hayoon jiyu yuna chaewon eunji hyejin jihye
soyeon taeyang hyun joon woo seung jin young anh minh linh thanh huong lan
trang ngoc duc tuan hung quang khanh phuong thao vy nam long tien dat kiet
bao hieu thu ha thuy trinh angelo carlo rico jericho jasmine rhea bea
kristine budi agus dewi sri putri putra ayu rizky fajar dimas bayu rina wati
ratna siti adi andi wahyu yogi indra hendra somchai niran anong ploy mali
nok kwame kofi kojo ama akua abena yaw kwaku adwoa efua chidi chinedu emeka
obinna ikenna nnamdi uche ngozi chioma adaeze amaka ifeoma nkechi tunde seun
femi kemi bola funmi tolu dayo wale yemi ayo lanre kunle tayo zola thabo
sipho themba lwazi nomsa thandi zanele lindiwe palesa lerato mpho kagiso
amara zuri imani baraka juma rehema neema halima amani jabari kamau wanjiru
njeri wambui achieng otieno ochieng abebe tsegaye mekdes selam liya dawit
yonas noam itai yonatan ariel eitan omer tal shira noa yael tamar avi eli
nikos giorgos yannis kostas dimitris christos panagiotis eleni katerina
despina ioanna vasiliki alexandros stavros mikey jonny benny sammy nicky
freddy izzy lizzy maddie addie kenzie mackenzie liv livvy allie ally gigi
lulu coco jojo
`)

// americasFirstNames lists the first names above that are common in English- and Spanish-speaking countries (560).
var americasFirstNames = strings.Fields(`
james john robert michael william david richard joseph thomas charles
christopher daniel matthew anthony mark donald steven paul andrew joshua
kenneth kevin brian george timothy ronald edward jason jeffrey ryan jacob
gary nicholas eric jonathan stephen larry justin scott brandon benjamin
samuel gregory alexander frank patrick raymond jack dennis jerry tyler aaron
adam nathan henry douglas zachary peter kyle noah ethan jeremy walter
christian keith roger terry austin sean gerald carl harold dylan arthur
lawrence jordan jesse bryan billy bruce gabriel joe logan alan albert wayne
randy vincent mason roy ralph bobby russell bradley philip eugene liam
oliver lucas aiden jayden carter owen luke isaac caleb hunter connor landon
evan colton cameron chase blake cole wyatt levi hudson grayson easton jaxon
asher lincoln miles leo max oscar harry charlie alfie freddie archie theo
toby finley reggie teddy louie rory ollie jamie callum declan kieran jake
josh tom ben dan sam alex chris mike matt nick tony steve rob will ed ted
jim tim andy danny tommy jimmy johnny joey kenny ricky jeff greg brad derek
travis troy shane dustin marcus darren trevor spencer seth grant dean neil
ross craig stuart graham gavin colin martin simon ian nigel barry trent reid
brett dale glenn clay beau jace kai ezra silas jasper felix hugo arlo milo
otis rowan finn ronan elliot emmett everett beckett brody bryce tristan
maddox ryder rhys harvey jenson zach nate gabe drew mary patricia jennifer
linda elizabeth barbara susan jessica sarah karen lisa nancy betty sandra
margaret ashley kimberly emily donna michelle carol amanda melissa deborah
stephanie dorothy rebecca sharon laura cynthia amy kathleen angela shirley
brenda emma anna pamela nicole samantha katherine christine helen debra
rachel carolyn janet maria catherine heather diane olivia julie joyce
victoria ruth virginia lauren kelly christina joan evelyn judith andrea
hannah megan cheryl jacqueline martha madison teresa gloria sara janice ann
kathryn abigail sophia frances jean alice judy isabella julia grace amber
denise danielle marilyn beverly charlotte natalie theresa diana brittany
doris kayla alexis lori marie ava mia amelia harper ella chloe lily zoe leah
aubrey addison layla scarlett aria riley nora hazel violet aurora savannah
audrey brooklyn bella claire skylar lucy paisley everly caroline nova
genesis emilia kennedy maya willow kinsley naomi aaliyah elena sadie ariana
allison gabriella madelyn cora ruby eva serenity autumn adeline hailey
gianna valentina isla eliana quinn nevaeh ivy piper lydia alexa josephine
emery delilah vivian stella clara freya poppy daisy evie rosie phoebe millie
florence matilda elsie imogen esme ellie holly jade katie kate jess becky
liz beth kat jen jenny molly abby gemma hayley tilly nell iris ada edith
mabel maisie lottie connie orla niamh siobhan aoife ciara saoirse juan jose
luis miguel carlos jorge pedro manuel francisco antonio javier diego pablo
mateo santiago sebastian nicolas alejandro fernando ricardo eduardo rafael
andres sergio raul alberto roberto hector ruben emilio joaquin ignacio
gonzalo cristian julio cesar marco tomas felipe gustavo rodrigo leonardo
bruno thiago enzo matias benicio santino valentino maximo lorenzo agustin
martina lucia valeria camila sofia daniela gabriela mariana fernanda paula
natalia carolina alejandra ximena renata regina antonella catalina florencia
agustina julieta luciana abril milagros rocio guadalupe carmen rosa ana
beatriz ines pilar marta cristina raquel silvia monica veronica lorena
adriana juliana leticia larissa bianca mikey jonny benny sammy nicky freddy
izzy lizzy maddie addie kenzie mackenzie liv livvy allie ally gigi lulu coco
jojo
`)

// commonFirstNames lists the most common first names and nicknames in English-speaking countries, for telling plain usernames like johnkevin (180).
var commonFirstNames = strings.Fields(`
james john robert michael william david richard joseph thomas charles
christopher daniel matthew anthony mark donald steven paul andrew joshua
kevin brian george timothy edward jason ryan jacob eric justin scott brandon
benjamin samuel alexander frank patrick jack tyler aaron adam nathan henry
zachary kyle noah liam oliver elijah lucas mason logan ethan aiden jayden
luke owen dylan caleb isaac gabriel carter wyatt jackson levi hunter austin
connor cameron evan jordan chase cole ian max leo jake josh mike matt chris
nick alex sam ben dan tom joe will tony jim steve dave jose luis carlos juan
mary patricia jennifer linda elizabeth barbara susan jessica sarah karen
lisa nancy margaret ashley emily michelle amanda melissa stephanie rebecca
laura amy angela emma anna nicole samantha katherine rachel heather maria
olivia julia victoria hannah grace abigail madison chloe sophia isabella ava
mia ella lily natalie zoe addison brooklyn avery aubrey leah hailey kayla
alyssa lauren taylor megan brianna morgan destiny jasmine sydney kaitlyn
alexis savannah riley allison ellie harper evelyn scarlett aria layla nora
kate katie jenny beth becky abby maddie amber
`)

// commonLastNames lists the most common US last names (150).
var commonLastNames = strings.Fields(`
smith johnson williams brown jones garcia miller davis rodriguez martinez
hernandez lopez gonzalez wilson anderson thomas taylor moore jackson martin
lee perez thompson white harris sanchez clark ramirez lewis robinson walker
young allen king wright scott torres nguyen hill flores green adams nelson
baker hall rivera campbell mitchell carter roberts gomez phillips evans
turner diaz parker cruz edwards collins reyes stewart morris morales murphy
cook rogers gutierrez ortiz morgan cooper peterson bailey reed kelly howard
ramos kim cox ward richardson watson brooks chavez wood james bennett gray
mendoza ruiz hughes price alvarez castillo sanders patel myers long ross
foster jimenez powell jenkins perry russell sullivan bell coleman butler
henderson barnes fisher vasquez simmons graham murray ford hamilton wallace
shaw gordon burns kennedy griffin west cole hayes chapman ellis stevens
tucker marshall owens harrison fernandez mcdonald woods washington palmer
lawrence black mills grant knight rose stone hawkins dunn perkins hudson
spencer
`)

// rareLastNames lists less common last names from many countries (1377).
var rareLastNames = strings.Fields(`
gardner stephens payne pierce berry matthews arnold wagner willis ray
watkins olson carroll duncan snyder hart cunningham bradley lane andrews
harper fox riley armstrong carpenter weaver greene elliott sims austin
peters kelley franklin lawson fields schmidt carr wheeler oliver montgomery
richards williamson johnston banks meyer bishop mccoy howell morrison hansen
garza harvey little burton stanley george jacobs reid fuller lynch dean
gilbert romero welch larson frazier burke hanson day moreno bowman medina
fowler brewer hoffman carlson silva pearson holland douglas fleming jensen
vargas byrd davidson hopkins may terry herrera wade soto walters curtis neal
caldwell lowe jennings barnett graves horton shelton barrett obrien castro
sutton gregory mckinney lucas miles craig chambers holt lambert fletcher
watts bates hale rhodes pena beck newman haynes mcdaniel mendez bush vaughn
parks dawson santiago norris hardy love steele curry powers schultz barker
guzman page munoz ball keller chandler weber leonard walsh lyons ramsey
wolfe schneider mullins benson sharp bowen barber cummings hines baldwin
griffith valdez hubbard salazar reeves warner stevenson burgess santos tate
cross garner mann mack moss thornton mcgee farmer delgado aguilar vega
glover manning cohen harmon rodgers robbins newton todd blair higgins ingram
reese cannon strickland townsend potter goodwin walton rowe hampton ortega
patton swanson francis goodman maldonado yates becker erickson hodges rios
conner adkins webster norman malone hammond flowers cobb moody quinn blake
maxwell pope floyd osborne mccarthy guerrero lindsey estrada sandoval gibbs
gross fitzgerald stokes doyle sherman saunders wise colon gill alvarado
greer padilla waters nunez ballard schwartz mcbride houston christensen
klein pratt briggs parsons mclaughlin zimmerman french buchanan moran
copeland pittman brady mccormick holloway brock poole frank logan owen bass
marsh drake wong jefferson morton abbott sparks norton huff clayton massey
lloyd figueroa carson bowers roberson barton tran lamb harrington casey
boone cortez clarke mathis singleton wilkins cain underwood hogan mckenzie
collier luna phelps mcguire allison bridges wilkerson nash summers atkins
muller fischer schulz hoffmann koch richter wolf schroder neumann schwarz
zimmermann braun kruger hofmann hartmann lange werner krause lehmann kohler
maier rossi russo ferrari esposito bianchi romano colombo ricci marino greco
bruno gallo conti deluca costa giordano mancini rizzo lombardi moretti
barbieri fontana santoro mariani rinaldi caruso ferrara galli martini leone
longo gentile martinelli vitale lombardo serra coppola desantis marchetti
parisi villa conte ferraro ferri fabbri bianco marini grasso valentini
messina sala gatti pellegrini palumbo sanna farina rizzi monti cattaneo
morelli amato silvestri mazza testa grassi carbone giuliani benedetti barone
rossetti caputo montanari guerra palmieri bernardi martino fiore ferretti
bellini basile riva donati piras vitali battaglia sartori neri costantini
milani pagano ruggiero sorrentino orlando negri bernard dubois petit durand
leroy moreau laurent lefebvre michel bertrand roux fournier morel girard
andre lefevre mercier dupont bonnet legrand garnier faure rousseau blanc
guerin roussel perrin morin clement gauthier dumont fontaine chevalier robin
masson gerard boyer denis lemaire duval joly gautier roche noel meunier
marchand dufour blanchard barbier brun dumas brunet schmitt leroux colin
renard arnaud rolland caron aubert giraud leclerc vidal bourgeois renaud
lemoine picard gaillard philippe leclercq lacroix fabre dupuis alonso
navarro dominguez vazquez gil serrano blanco molina suarez rubio marin sanz
iglesias garrido cortes lozano cano prieto calvo gallego leon marquez
cabrera campos fuentes carrasco diez caballero nieto pascual santana herrero
lorenzo montero hidalgo gimenez ibanez ferrer duran benitez mora vicente
arias carmona crespo roman pastor saez velasco moya soler parra esteban
bravo gallardo rojas oliveira souza sousa pereira ferreira rodrigues almeida
alves carvalho gomes martins araujo ribeiro lima barbosa rocha dias
nascimento andrade moreira nunes marques machado mendes freitas cardoso
goncalves teixeira correia pinto lopes cunha reis monteiro fonseca batista
melo vieira cavalcanti devries jansen dejong bakker visser smit meijer boer
mulder bos vos hendriks dekker brouwer dijkstra nielsen pedersen andersen
larsen sorensen rasmussen jorgensen petersen madsen kristensen olsen thomsen
poulsen johansson karlsson nilsson eriksson larsson olsson persson svensson
gustafsson pettersson jonsson lindberg lindqvist berg holm lund strom
virtanen korhonen nieminen makinen laine heikkinen koskinen jarvinen nowak
kowalski wisniewski wojcik kowalczyk kaminski lewandowski zielinski
szymanski wozniak dabrowski kozlowski jankowski mazur kwiatkowski krawczyk
piotrowski grabowski nowakowski pawlowski michalski ivanov petrov sidorov
smirnov kuznetsov popov sokolov lebedev kozlov novikov morozov volkov
solovyov vasiliev zaitsev pavlov semenov golubev vinogradov bogdanov
vorobyov fedorov mikhailov belyaev tarasov belov komarov orlov kiselev
makarov andreev kovalenko bondarenko tkachenko shevchenko kravchenko novak
horvat kovac babic maric jovanovic petrovic nikolic markovic djordjevic
stojanovic ilic stankovic pavlovic dvorak svoboda novotny prochazka cerny
vesely horak nemec wang li zhang liu chen yang huang zhao wu zhou xu sun ma
zhu hu guo he lin gao luo zheng liang xie song tang han feng deng cao peng
zeng xiao tian dong pan yuan cai jiang yu du ye cheng wei su lu ding ren
shen yao fu zhong cui tan liao fan sato suzuki takahashi tanaka watanabe ito
yamamoto nakamura kobayashi kato yoshida yamada sasaki yamaguchi matsumoto
inoue kimura hayashi shimizu yamazaki mori abe ikeda hashimoto yamashita
ishikawa nakajima maeda fujita ogawa goto okada hasegawa murakami kondo
ishii saito sakamoto endo aoki fujii nishimura fukuda ota miura fujiwara
okamoto matsuda nakagawa park choi jung kang cho yoon jang lim oh seo shin
kwon hwang ahn jeon hong yoo ko moon son bae baek heo nam le pham hoang
huynh phan vu vo dang bui do ho ngo duong ly singh kumar sharma gupta shah
khan reddy rao iyer nair menon pillai das bose chatterjee banerjee mukherjee
ghosh sen dutta chopra kapoor malhotra mehta joshi desai jain agarwal verma
yadav mishra pandey tiwari saxena srivastava chauhan thakur rajput naidu
shetty hegde kulkarni patil pawar jadhav shinde bhatt trivedi dave sheth
parekh ahmed hussain chaudhry qureshi siddiqui sheikh mirza baig awan rana
raza abbasi hashmi rizvi naqvi zaidi haddad khoury mansour saleh aziz rahman
hamdan yilmaz kaya demir sahin celik yildiz yildirim ozturk aydin ozdemir
arslan dogan kilic aslan cetin kara koc kurt ozkan simsek polat korkmaz
okafor okonkwo adeyemi adebayo ogunleye mensah owusu asante boateng osei
acheampong nkosi dlamini ndlovu khumalo mokoena mahlangu mwangi odhiambo
kariuki njoroge onyango kiprop cheruiyot tesfaye bekele haile girma alemu
diallo traore coulibaly keita toure camara sow ndiaye diop fall oconnor
byrne gallagher mcloughlin brennan macdonald mackenzie macleod fraser
robertson paterson mclean mcgregor levi mizrahi peretz biton friedman
shapiro katz goldberg rosenberg weiss papadopoulos georgiou nikolaou
dimitriou papadakis konstantinou ashby ashworth atwood bancroft barlow
barnaby beckett belcher benning berkley blackwood blakely bramley brannon
bristow calloway carver cassidy chadwick channing chapple colby conway
corbett crowder crowley dalton danvers darby delaney dempsey denholm devlin
dorsey drummond dudley dunbar dunmore eastwood ellery ellison everly fairley
farrow fenwick finch fitch flint forde foxworth gable galloway garland
garrity gentry goddard goodall gorman granger greaves hadley halsey hammett
hanley harding harlow hartley haskell hatfield hawley hayward heller hendry
hensley hobbs holbrook hollis hopper horne howland hurley hutchins ingalls
irving jarvis jeffries kearney keating kendrick kenney kershaw kimball
kinsey kirby knox lacey langley larkin latham layton leland lester linden
lockwood lowry lyle macaulay maddox malloy marlow marston mayfield
mcallister mccabe mccall mcgrath mckee mcnally merritt milburn millard
monroe mosley mulligan norwood oakley ogden osgood oswald paget parrish
paxton pemberton penrose percy pickett platt prescott pritchard quigley
radford ramsay rawlings redding renfro rigby ripley rooney rourke rowley
rutledge sadler sampson sawyer seaton sexton shelby sheridan shipley
sinclair slater somers spalding stafford stoddard stroud sutherland talbot
tanner tatum templeton thorne tilley toomey trask truman tully upton vance
vickers wakefield waldron warwick weatherby wendell wexler whitaker whitley
wilcox winslow wolcott woodard wren yardley yoder arsenault beaulieu
belanger bouchard boucher cloutier desjardins gagnon giroux lachance lalonde
leblanc lemieux levesque pelletier poirier savard thibault tremblay vachon
ouellette paquette fortin cormier theriault lavoie gosselin bergeron
albrecht baumann bergmann brandt eckert engel gerber haas hahn holtz kessler
kraft kuhn lorenz mayer metzger pfeiffer reinhardt schafer schilling seidel
vogel voss winkler ziegler brenner fuchs lindner bonanno capello napoli
tedesco valenti santangelo aguirre barrera bustos cardenas castaneda
cervantes cisneros escobar espinoza galvan ibarra leyva macias mejia montoya
navarrete ochoa olvera orozco pacheco palacios quintero rangel robles
salinas sepulveda solis tapia trevino urbina valencia villarreal zamora
zavala arellano beltran cordero bergstrom dahl ekman engstrom falk hagen
lindgren lundqvist nygaard sandberg sjoberg strand wahl popescu radu stanek
kovacs szabo nagy toth varga callahan carney cleary connolly costello cullen
donnelly egan fahey finnegan flanagan flynn hanlon hennessy kavanagh keane
kinsella mahoney mcnamara molloy moriarty mullen nagle phelan quinlan
riordan scanlon sheehan tierney whelan
`)
