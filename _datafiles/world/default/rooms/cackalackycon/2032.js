// Trivia Stage - Vic Vandal's Hacker Trivia
// Disembodied voice asks questions, players answer for gold

const REWARD_GOLD = 10;

const TRIVIA = [
    {
        q: "In the movie 'Hackers', what is the name of the virus that threatens to capsize oil tankers?",
        a: ["da vinci", "davinci", "the da vinci virus"],
        hint: "Think Renaissance artist..."
    },
    {
        q: "What year was DEFCON first held?",
        a: ["1993", "93"],
        hint: "Early 90s, Clinton era..."
    },
    {
        q: "In 'WarGames', what game does the WOPR want to play?",
        a: ["global thermonuclear war", "thermonuclear war"],
        hint: "Not tic-tac-toe..."
    },
    {
        q: "What does SQL stand for?",
        a: ["structured query language"],
        hint: "It's very... structured."
    },
    {
        q: "Who wrote the Morris Worm, the first major internet worm?",
        a: ["robert morris", "robert tappan morris", "morris"],
        hint: "His last name IS the worm's name."
    },
    {
        q: "What hacker magazine was published from 1984 to 2018?",
        a: ["2600", "2600 magazine", "2600 the hacker quarterly"],
        hint: "It's a frequency in hertz..."
    },
    {
        q: "In 'The Matrix', what color is the pill Neo takes?",
        a: ["red", "the red pill", "red pill"],
        hint: "Not blue..."
    },
    {
        q: "What port does SSH typically run on?",
        a: ["22", "port 22"],
        hint: "A low number, very secure."
    },
    {
        q: "What was the name of Kevin Mitnick's book about social engineering?",
        a: ["the art of deception", "art of deception"],
        hint: "It's an art form..."
    },
    {
        q: "What does the 'S' in HTTPS stand for?",
        a: ["secure"],
        hint: "It makes things... safer."
    },
    {
        q: "In 'Sneakers', what does the anagram 'SETEC ASTRONOMY' solve to?",
        a: ["too many secrets", "too many secret"],
        hint: "It reveals something hidden..."
    },
    {
        q: "What year was the first iPhone jailbreak released?",
        a: ["2007", "07"],
        hint: "Same year the iPhone launched..."
    },
    {
        q: "What language was the first computer virus, 'Creeper', written in?",
        a: ["assembly", "asm", "assembler"],
        hint: "Very low level..."
    },
    {
        q: "What is the default port for HTTP?",
        a: ["80", "port 80"],
        hint: "A nice round number..."
    },
    {
        q: "Who founded WikiLeaks?",
        a: ["julian assange", "assange"],
        hint: "Currently in legal troubles..."
    }
];

var currentQuestion = null;
var questionAsked = false;
var roundsSinceQuestion = 0;
var hintGiven = false;

function onIdle(room) {
    roundsSinceQuestion++;
    
    // Ask a new question every ~8 rounds if none active (~32 seconds)
    if (!questionAsked && roundsSinceQuestion >= 8) {
        askQuestion(room);
        return true;
    }
    
    // Give hint after 10 rounds
    if (questionAsked && !hintGiven && roundsSinceQuestion >= 10) {
        SendRoomMessage(room.RoomId(), "");
        SendRoomMessage(room.RoomId(), "<ansi fg=\"yellow\">Vic Vandal's voice echoes: \"Need a hint? " + currentQuestion.hint + "\"</ansi>");
        hintGiven = true;
        return true;
    }
    
    // Time out after 20 rounds
    if (questionAsked && roundsSinceQuestion >= 25) {
        SendRoomMessage(room.RoomId(), "");
        SendRoomMessage(room.RoomId(), "<ansi fg=\"red\">Vic Vandal sighs: \"Time's up! The answer was: " + currentQuestion.a[0] + "\"</ansi>");
        SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">\"Better luck next round, hackers!\"</ansi>");
        questionAsked = false;
        currentQuestion = null;
        hintGiven = false;
        roundsSinceQuestion = 0;
        return true;
    }
    
    // Random Vic commentary
    if (UtilGetRoundNumber() % 8 == 0 && !questionAsked) {
        var quips = [
            "",
            "<ansi fg=\"cyan\">Vic Vandal's voice echoes: \"Who's ready for some trivia?\"</ansi>",
            "<ansi fg=\"cyan\">A disembodied voice chuckles somewhere in the darkness.</ansi>",
            "",
            ""
        ];
        var idx = UtilDiceRoll(1, quips.length) - 1;
        if (quips[idx] != "") {
            SendRoomMessage(room.RoomId(), quips[idx]);
            return true;
        }
    }
    
    return false;
}

function askQuestion(room) {
    var idx = UtilDiceRoll(1, TRIVIA.length) - 1;
    currentQuestion = TRIVIA[idx];
    questionAsked = true;
    roundsSinceQuestion = 0;
    hintGiven = false;
    
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan-bold\">═══════════════════════════════════════════</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"yellow-bold\">  VIC VANDAL'S VOICE BOOMS FROM THE SHADOWS:</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan-bold\">═══════════════════════════════════════════</ansi>");
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"white-bold\">\"" + currentQuestion.q + "\"</ansi>");
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"green\">Type ANSWER followed by your answer to win " + REWARD_GOLD + " gold!</ansi>");
    SendRoomMessage(room.RoomId(), "");
}

function onCommand_say(rest, user, room) {
    if (!questionAsked || currentQuestion == null) {
        return false;
    }
    
    var answer = rest.toLowerCase().trim();
    
    for (var i = 0; i < currentQuestion.a.length; i++) {
        if (answer.includes(currentQuestion.a[i])) {
            // Correct!
            SendRoomMessage(room.RoomId(), "");
            SendRoomMessage(room.RoomId(), "<ansi fg=\"green-bold\">★ CORRECT! ★</ansi>");
            SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">Vic Vandal exclaims: \"" + user.GetCharacterName(false) + " NAILS IT!\"</ansi>");
            SendRoomMessage(room.RoomId(), "");
            
            user.AddGold(REWARD_GOLD);
            user.SendText("<ansi fg=\"yellow\">You receive " + REWARD_GOLD + " gold!</ansi>");
            
            questionAsked = false;
            currentQuestion = null;
            hintGiven = false;
            roundsSinceQuestion = 0;
            
            return false; // Let the say still go through
        }
    }
    
    return false; // Wrong answer, but don't block the say
}

function onCommand_answer(rest, user, room) {
    // Alias for answering
    if (!questionAsked) {
        user.SendText("<ansi fg=\"yellow\">Vic Vandal's voice whispers: \"No question active right now. Wait for the next one!\"</ansi>");
        return true;
    }
    
    // Process as if they said it
    return onCommand_say(rest, user, room);
}
