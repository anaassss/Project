package knowledge

import "strings"

// Word lists the patterns draw from. Names are common first and last names
// from many countries; none of this is anyone's actual username.

var firstNames = strings.Fields(`
james john robert michael william david richard joseph thomas charles
christopher daniel matthew anthony mark steven paul andrew joshua kevin
brian george edward ryan jacob nicholas eric jonathan justin brandon
samuel benjamin alexander patrick jack tyler aaron adam nathan kyle
ethan noah liam lucas mason logan oliver elijah aiden carter owen luke
dylan gabriel isaac caleb henry sebastian leo max oscar harry charlie
alfie freddie archie theo arthur mohammed ahmed ali omar hassan yusuf
ibrahim raj arjun rahul amit vikram rohan aditya kiran wei jun hao
kenji hiroshi takumi carlos juan jose luis miguel diego pablo mateo
santiago marco luca giovanni pierre louis hugo lukas felix jonas lars
sven erik ivan dmitri alexei mary patricia jennifer linda elizabeth
barbara susan jessica sarah karen lisa nancy sandra ashley emily
michelle amanda melissa stephanie rebecca laura emma olivia ava sophia
isabella mia amelia harper evelyn abigail ella grace chloe lily zoe
hannah natalie leah anna maria sofia lucia valentina camila elena
julia clara ines fatima aisha zainab maryam priya ananya divya pooja
neha mei yuki sakura hana jisoo nora freya ingrid astrid olga natasha
katie kate amy ellie ruby holly megan lauren jade sam alex jordan
taylor casey morgan riley jamie mike chris matt nick tom ben dan joe
jake josh tony steve jen jess becky liz beth kat
`)

var lastNames = strings.Fields(`
smith johnson williams brown jones garcia miller davis rodriguez
martinez hernandez lopez gonzalez wilson anderson thomas taylor moore
jackson martin lee perez thompson white harris sanchez clark ramirez
lewis robinson walker young allen king wright scott torres nguyen hill
flores green adams nelson baker hall rivera campbell mitchell carter
roberts gomez phillips evans turner diaz parker cruz edwards collins
reyes stewart morris morales murphy cook rogers gutierrez ortiz morgan
cooper peterson bailey reed kelly howard ramos cox ward richardson
watson brooks chavez wood bennett gray mendoza ruiz hughes price
alvarez castillo sanders patel myers long ross foster jimenez powell
jenkins perry russell sullivan bell coleman butler henderson barnes
fisher vasquez simmons graham murray ford hamilton wallace shaw gordon
burns kennedy singh kumar sharma gupta shah khan hussain chen wang
zhang liu yang huang zhao zhou tanaka suzuki sato takahashi watanabe
park choi jung kang cho muller schmidt schneider fischer weber meyer
wagner becker hoffmann rossi russo ferrari esposito bianchi romano
dubois bernard moreau laurent silva santos oliveira souza pereira
costa ferreira novak kowalski nowak ivanov petrov jensen hansen
nielsen larsen andersen johansson karlsson nilsson obrien walsh byrne
oconnor
`)

var adjectives = strings.Fields(`
happy lazy silent dark bright crazy cosmic golden silver wild swift
brave lucky sleepy sneaky mighty tiny fuzzy frosty sunny stormy misty
neon pixel cyber retro mystic lunar solar royal noble rapid quiet
loud angry chill cool epic hyper super mega ultra grumpy jolly witty
clever fierce savage gentle humble shady spicy salty sweet sour bitter
fancy funky groovy jazzy lost wandering hidden secret ancient electric
atomic toxic frozen burning broken iron steel crystal velvet scarlet
crimson azure emerald amber ivory midnight twilight northern southern
wicked brutal rogue tactical stealthy hungry curious magic
sonic turbo nitro rusty dusty dizzy bouncy crispy cheesy fluffy
`)

var nouns = strings.Fields(`
wolf fox panda tiger dragon phoenix raven hawk falcon eagle owl bear
lion shark viper cobra ninja samurai knight wizard ranger hunter pilot
rider runner gamer coder artist dreamer storm thunder shadow star moon
sun comet nova ocean river forest cloud rain snow frost flame blaze
ember spark pixel byte code ghost spirit soul blade arrow shield crown
rose lily cookie muffin taco pizza noodle coffee mango peach cherry
berry honey penguin otter koala bunny kitten puppy turtle dolphin
whale octopus panther jaguar lynx cheetah mustang stallion hornet
mantis scorpion kraken hydra golem titan giant ghoul reaper slayer
seeker keeper walker breaker maker master lord king queen boss chief
captain pirate outlaw bandit nomad drifter voyager explorer legend
hero champion rebel knightmare sniper tank medic scout squad legion
vortex nebula galaxy orbit rocket meteor laser plasma quasar
glitch vector matrix circuit widget gadget robot android cyborg
potato pickle waffle pancake bagel donut burrito nacho biscuit
`)

var hobbies = strings.Fields(`
gaming games plays codes draws art music beats bakes cooks travels
reads writes lifts runs fit photo films vlogs world zone hub life
vibes daily official
`)

var namePrefixes = strings.Fields(`its im iam the real just hey mr ms`)

var channelSuffixes = strings.Fields(`TV YT Live HD Plays`)
