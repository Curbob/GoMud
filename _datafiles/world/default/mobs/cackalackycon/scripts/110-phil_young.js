// Phil Young - Canadian Mainframe Karaoke Host
// Alternates between singing and mainframe facts

var songMode = false;
var verseCount = 0;

const SONGS = [
    {
        title: "Don't Stop Believin'",
        verses: [
            "emote grabs the mic dramatically.",
            "shout 🎵 Just a small town girl, livin' in a lonely world! 🎵",
            "shout 🎵 She took the midnight train goin' anywhere! 🎵",
            "emote points at the crowd."
        ]
    },
    {
        title: "Bohemian Rhapsody", 
        verses: [
            "emote closes his eyes solemnly.",
            "shout 🎵 Is this the real life? Is this just fantasy? 🎵",
            "shout 🎵 Caught in a landslide, no escape from reality! 🎵",
            "emote drops to his knees for emphasis."
        ]
    },
    {
        title: "Never Gonna Give You Up",
        verses: [
            "emote winks at the audience.",
            "shout 🎵 Never gonna give you up! Never gonna let you down! 🎵",
            "shout 🎵 Never gonna run around and desert you! 🎵",
            "emote does a little dance."
        ]
    },
    {
        title: "Sweet Caroline",
        verses: [
            "emote raises his hands to get the crowd going.",
            "shout 🎵 Sweet Caroline! 🎵",
            "say Everyone! BAH BAH BAH!",
            "shout 🎵 Good times never seemed so good! 🎵"
        ]
    }
];

const MAINFRAME_FACTS = [
    "say Did you know the IBM z16 can process 300 billion inferences per day? Per DAY, eh!",
    "say COBOL processes 95% of ATM transactions. That's not legacy, that's LEGENDARY.",
    "say The mainframe has been 'dying' for 40 years. Still here. Still processing your bank transactions.",
    "say JCL is beautiful. You just don't understand it yet.",
    "say Fun fact: The original IBM System/360 had less memory than your badge. But it ran ACTUAL businesses.",
    "say CICS has been transaction processing since 1969. Kubernetes is cute though.",
    "say Some z/OS systems have uptimes measured in YEARS. When's the last time your cloud did that, eh?",
    "say DB2 on mainframe? *chef's kiss* Perfection.",
    "say People say mainframes are expensive. You know what's expensive? Downtime.",
    "say RACF security has been rock solid since 1976. Your OAuth implementation... not so much.",
    "say The 'green screen' isn't a limitation, it's an AESTHETIC.",
    "say Back in my day... wait, I'm 28. But SPIRITUALLY I'm from the mainframe era."
];

const CANADIAN_PHRASES = [
    "eh",
    "Sorry, got carried away there, eh.",
    "Oh, sorry aboot that.",
    "Beauty!",
    "That's what I appreciates about mainframes."
];

function onIdle(mob, room) {
    if (UtilGetRoundNumber() % 4 != 0) {
        return false;
    }
    
    // 40% chance to sing, 40% mainframe fact, 20% nothing
    var roll = UtilDiceRoll(1, 10);
    
    if (roll <= 4) {
        // Sing a song
        if (!songMode) {
            var songIdx = UtilDiceRoll(1, SONGS.length) - 1;
            var song = SONGS[songIdx];
            mob.Command("say Okay everyone, this next one is '" + song.title + "'!");
            songMode = true;
            verseCount = 0;
        } else {
            var songIdx = UtilDiceRoll(1, SONGS.length) - 1;
            var song = SONGS[songIdx];
            if (verseCount < song.verses.length) {
                mob.Command(song.verses[verseCount]);
                verseCount++;
            } else {
                mob.Command("emote takes a bow.");
                var canadian = CANADIAN_PHRASES[UtilDiceRoll(1, CANADIAN_PHRASES.length) - 1];
                mob.Command("say " + canadian, 1);
                songMode = false;
                verseCount = 0;
            }
        }
        return true;
    } else if (roll <= 8) {
        // Mainframe fact
        if (!songMode) {
            var factIdx = UtilDiceRoll(1, MAINFRAME_FACTS.length) - 1;
            mob.Command(MAINFRAME_FACTS[factIdx]);
            return true;
        }
    }
    
    return false;
}

function onGreet(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    mob.Command("say Oh hey there! Welcome to karaoke night, eh!");
    mob.Command("say You wanna hear about mainframes? Or maybe sing a song? Or both?", 1.5);
    
    return true;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    var ask = eventDetails.askText.toLowerCase();
    
    if (ask.includes("mainframe") || ask.includes("ibm") || ask.includes("cobol")) {
        mob.Command("emote 's eyes light up with pure joy.");
        mob.Command("say Oh, you want to talk mainframes?! Finally! Someone gets it!", 1);
        mob.Command("say The IBM z16 is a thing of BEAUTY. 200+ processors, hardware AI acceleration...", 2);
        mob.Command("say Sorry, sorry. I get excited. It's just... they're so RELIABLE, eh?", 4);
        return true;
    }
    
    if (ask.includes("canada") || ask.includes("canadian")) {
        mob.Command("say Oh yeah, I'm from Canada! Toronto area. Great mainframe community up there.");
        mob.Command("say The cold keeps the data centers cool, eh!", 1.5);
        return true;
    }
    
    if (ask.includes("sing") || ask.includes("song") || ask.includes("karaoke")) {
        mob.Command("say You wanna sing? The mic's right here! Just step up!");
        mob.Command("say I do requests too. Mostly Journey. And Rush - they're Canadian, you know!", 1.5);
        return true;
    }
    
    if (ask.includes("cloud") || ask.includes("kubernetes") || ask.includes("container")) {
        mob.Command("emote sighs deeply.");
        mob.Command("say Look, the cloud is fine. Fine! But a mainframe...", 1);
        mob.Command("say A mainframe is FOREVER. Sorry, I don't mean to be harsh, eh.", 2);
        return true;
    }
    
    if (ask.includes("age") || ask.includes("young") || ask.includes("old")) {
        mob.Command("say I'm 28! Yes, I know I'm probably the only mainframe fan under 85.");
        mob.Command("say Someone's gotta keep the faith alive, eh?", 1.5);
        return true;
    }
    
    mob.Command("say That's a great question! But have you considered... mainframes?");
    return true;
}
