package knowledge

import "strings"

// Word lists the patterns draw from. None of this is anyone's actual
// username.

// firstNames lists common first names and nicknames from many countries (1254).
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
bao my hieu thu ha thuy trinh angelo carlo rico jericho jasmine rhea bea
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

// lastNames lists common last names from many countries (1182).
var lastNames = strings.Fields(`
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
spencer gardner stephens payne pierce berry matthews arnold wagner willis
ray watkins olson carroll duncan snyder hart cunningham bradley lane andrews
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
dimitriou papadakis konstantinou
`)

// gamingAdjectives lists adjectives gamers put in handles (178).
var gamingAdjectives = strings.Fields(`
silent dark bright crazy cosmic golden silver wild swift brave lucky sneaky
mighty frosty stormy misty neon pixel cyber retro mystic lunar solar royal
noble rapid savage fierce toxic frozen burning broken iron steel crystal
scarlet crimson azure emerald amber ivory midnight twilight wicked brutal
rogue tactical stealthy hungry electric atomic sonic turbo nitro hyper super
mega ultra epic legendary mythic ancient eternal immortal infinite phantom
chaotic deadly lethal vicious furious raging hidden secret lost fallen
cursed blessed holy unholy dire grim venomous radiant blazing flaming icy
stone obsidian cobalt chrome titanium quantum digital virtual binary glitchy
stellar galactic astral arcane primal feral alpha omega sigma prime elite
imperial masked hooded veiled wandering howling roaring soaring apex supreme
ultimate dizzy lazy sleepy grumpy salty spicy cheesy crispy fluffy fuzzy
tiny chill cool angry mad sly quick nimble clever cunning bold daring
reckless fearless ruthless relentless restless tireless heartless nameless
faceless endless shattered hollow rusty dusty bouncy funky jolly sour bitter
shady humble gentle loud quiet frantic manic sinister vengeful valiant
gallant loyal rabid feisty zesty sparky spooky creepy
`)

// gamingNouns lists nouns gamers put in handles: creatures, classes, weapons, space, weather, gems, food (382).
var gamingNouns = strings.Fields(`
wolf fox panda tiger dragon phoenix raven hawk falcon eagle owl bear lion
shark viper cobra ninja samurai knight wizard ranger hunter pilot rider
runner gamer coder storm thunder shadow star moon sun comet nova blade arrow
shield crown ghost spirit soul reaper slayer seeker keeper walker breaker
maker master lord king queen boss chief captain pirate outlaw bandit nomad
drifter voyager explorer legend hero champion rebel sniper tank medic scout
squad legion vortex nebula galaxy orbit rocket meteor laser plasma quasar
glitch vector matrix circuit robot android cyborg titan giant golem hydra
kraken ghoul demon wraith specter banshee valkyrie viking spartan gladiator
warrior paladin berserker assassin mage sorcerer warlock necromancer druid
shaman monk templar crusader sentinel guardian warden archer gunner striker
raider marauder invader predator stalker tracker trapper prowler lurker
beast monster fiend brute behemoth leviathan colossus juggernaut wyvern
basilisk chimera manticore minotaur cerberus pegasus unicorn sphinx kitsune
yeti mammoth rhino gorilla panther jaguar lynx cheetah leopard cougar
mustang stallion bull bison buffalo moose stag elk wolverine badger otter
ferret weasel hornet wasp scorpion mantis spider tarantula beetle condor
vulture osprey kestrel crow magpie serpent snake python anaconda mamba croc
gator raptor rex dino blizzard tornado hurricane typhoon cyclone tempest
thunderbolt lightning flash spark ember blaze inferno flame fire frost ice
glacier avalanche quake tremor tsunami volcano magma lava smoke mist fog
haze dusk dawn eclipse zenith horizon abyss void chasm rift nexus core pulse
surge shock static signal echo cipher code byte bit node kernel sword saber
katana dagger spear axe hammer mace scythe bullet rifle cannon missile armor
helm throne empire kingdom realm dominion citadel fortress bastion tower
castle bishop rook pawn ace joker spade diamond sapphire topaz opal pearl
quartz garnet mercury venus mars jupiter saturn neptune pluto orion sirius
pulsar asteroid satellite shuttle maverick jet racer engine piston gear mech
bot droid drone potato pickle waffle pancake bagel donut burrito nacho
biscuit taco pizza noodle cookie muffin mango peach cherry berry honey
coffee penguin koala bunny kitten puppy turtle dolphin whale octopus hamster
llama sloth narwhal walrus goose duck chicken frog toad lizard gecko squid
monkey cheese bean nugget burger bacon pepper pug cow pig hippo shrimp toast
noob meme doge cat dog fish bird chip candy sugar spice juice soda milk rice
pasta ramen sushi mochi
`)

// gamingTitles lists titles gamers put before a noun: SirWaffle, LordVortex (16).
var gamingTitles = strings.Fields(`
sir lord captain mr dr king queen prince baron duke general major agent
professor doctor chief
`)

// channelSuffixes lists streaming and channel suffixes: NovaTV, PixelYT (8).
var channelSuffixes = strings.Fields(`
TV YT TTV Live HD Plays GG Pro
`)

// socialWords lists words social media handles are made of: lunar.dreams, velvet_rose (245).
var socialWords = strings.Fields(`
angel aura blossom bloom bunny butterfly candy cherry cloud cotton daisy
dream dreamy fairy flower glitter glow heart honey kitty lavender lily love
lovely luna magic melody mint moon moonlight peach pearl petal pixie rose
sakura sky soft sparkle star sugar sunflower sunny sweet velvet vibes violet
wish golden latte matcha boba mocha vanilla caramel cocoa berry strawberry
blueberry lemon mango coconut papaya kiwi cupcake cookie cinnamon
marshmallow bubble bubbles sprinkle rainbow sunshine sunset sunrise ocean
wave waves sea shell coral lagoon island beach breeze meadow garden forest
fern ivy willow maple pine cedar moss river lake rain snow snowflake winter
summer autumn spring daydream wanderer wanderlust nomad soul spirit serene
calm cozy comfy chill mellow gentle quiet shy sleepy lazy cute pretty tiny
little mini muse poet poetry verse lyric song tune vinyl cassette polaroid
film camera lens canvas paint sketch doodle ink paper book novel story tea
chai croissant brunch toast cat kitten puppy pup fox deer fawn koala otter
panda bee ladybug dove swan owl moth firefly jellyfish starfish seahorse
dolphin whale turtle crystal gem jewel diamond opal moonstone stardust
cosmos galaxy nebula planet orbit comet venus celestial ethereal mystic
witch spell charm lucky clover rosy dewy misty foggy hazy dusty vintage
retro indie grunge punk goth pastel neon glossy silk satin lace linen denim
wild free happy lunar solar dreams stars clouds roses petals daisies hearts
angels skies moons blooms flowers wishes dreamer stargazer moonchild
sunkissed daydreamer honeybee wildflower bluebell buttercup
`)

// hobbies lists hobbies and trades people add to their names: sarahbakes, mike.codes (67).
var hobbies = strings.Fields(`
gaming games gamer playz plays anime live yt tv dance clips codes draws art
music beats bakes cooks travels reads writes lifts runs fit photo films
vlogs world zone hub life vibes daily official design designs studio creates
makes builds dev dances sings paints eats style fashion beauty makeup nails
fitness yoga rides skates surfs hikes climbs knits crafts garden plants
books tech tunes mixes edits shoots
`)

// socialPrefixes lists words people put before their names: mr.tobias, lil_zara, iamnoah (17).
var socialPrefixes = strings.Fields(`
mr mrs miss ms lil little big not my get iam its im the just real hey
`)

// nameSuffixes lists words people put after their names: leoboi, emmagirl (8).
var nameSuffixes = strings.Fields(`
boi boy girl man kid dude bro guy
`)

// relationWords lists family words in handles: noahsmama, daddy.omar (7).
var relationWords = strings.Fields(`
mom mum mama mommy dad daddy papa
`)

// funWords lists slang and exclamations in handles: yeet.lucas, zara.hello (19).
var funWords = strings.Fields(`
yeet oof lol idk bruh yolo hello hey meme epic cool boss legend vibe chill
lit hype wow omg
`)

// webSuffixes lists file and web endings on handles: lopez.io, nina.exe (8).
var webSuffixes = strings.Fields(`
com io co exe png www lol jpg
`)

// gamingPrefixes lists words gamers put before a word: mrwaffle, lilpanda, supertaco (12).
var gamingPrefixes = strings.Fields(`
mr lil little big super epic the its my not mega ultra
`)

// gamingSuffixes lists words gamers put after a word: tacoman, pandagirl, leoplayz (19).
var gamingSuffixes = strings.Fields(`
boi boy girl man gamer games playz plays pro king queen xx xd yt tv hd dude
bro master
`)
