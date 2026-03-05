// The Server Room - Portal Trigger Script
// Requires 4 key fragments to connect

const FROSTFANG_TOWN_SQUARE = 1;
const CRYPTO_FRAGMENT = 5100;
const HARDWARE_FRAGMENT = 5101;
const SOCIAL_FRAGMENT = 5102;
const CTF_FRAGMENT = 5103;
const NEWBIE_KIT = 100; // Fantasy starter gear

function onCommand_connect(rest, user, room) {
    attemptPortal(user, room);
    return true;
}

function onCommand_use(rest, user, room) {
    if (rest.toLowerCase().includes("computer") || 
        rest.toLowerCase().includes("machine") || 
        rest.toLowerCase().includes("terminal")) {
        attemptPortal(user, room);
        return true;
    }
    return false;
}

function onCommand_press(rest, user, room) {
    if (rest.toLowerCase().includes("key") || rest.toLowerCase().includes("enter")) {
        attemptPortal(user, room);
        return true;
    }
    return false;
}

function attemptPortal(user, room) {
    // Check for all fragments
    hasCrypto = user.HasItemId(CRYPTO_FRAGMENT);
    hasHardware = user.HasItemId(HARDWARE_FRAGMENT);
    hasSocial = user.HasItemId(SOCIAL_FRAGMENT);
    hasCTF = user.HasItemId(CTF_FRAGMENT);
    
    missing = [];
    if (!hasCrypto) missing.push("crypto");
    if (!hasHardware) missing.push("hardware");
    if (!hasSocial) missing.push("social");
    if (!hasCTF) missing.push("CTF");
    
    if (missing.length > 0) {
        user.SendText("");
        user.SendText("<ansi fg=\"cyan\">You press a key on the ancient keyboard...</ansi>");
        user.SendText("");
        user.SendText("<ansi fg=\"red\">ACCESS DENIED</ansi>");
        user.SendText("<ansi fg=\"red\">AUTHENTICATION FRAGMENTS REQUIRED</ansi>");
        user.SendText("");
        user.SendText("<ansi fg=\"yellow\">The screen flickers:</ansi>");
        user.SendText("<ansi fg=\"white\">\"Four keys open the way. Seek knowledge from:</ansi>");
        user.SendText("<ansi fg=\"white\"> - The cipher masters (Cipher Room)</ansi>");
        user.SendText("<ansi fg=\"white\"> - The hardware wizards (Hardware Hacking)</ansi>");
        user.SendText("<ansi fg=\"white\"> - The social engineers (SAV Lounge)</ansi>");
        user.SendText("<ansi fg=\"white\"> - The flag hunters (CTF Arena)\"</ansi>");
        user.SendText("");
        user.SendText("<ansi fg=\"red\">Missing fragments: " + missing.join(", ") + "</ansi>");
        user.SendText("");
        return;
    }
    
    // Has all fragments - trigger portal!
    triggerPortal(user, room);
}

function triggerPortal(user, room) {
    user.SendText("");
    user.SendText("<ansi fg=\"cyan\">You place all four key fragments on the keyboard...</ansi>");
    user.SendText("<ansi fg=\"lime\">They snap together, forming a complete access key!</ansi>");
    user.SendText("");
    
    room.SendText(
        "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> places four glowing fragments on the old machine's keyboard.",
        user.UserId()
    );
    
    // Dramatic transition sequence
    user.SendText("<ansi fg=\"green\">The CRT monitor blazes to life.</ansi>", 1);
    user.SendText("<ansi fg=\"yellow\">ACCESS GRANTED</ansi>", 2);
    user.SendText("<ansi fg=\"yellow\">INITIATING TRANSFER PROTOCOL...</ansi>", 3);
    user.SendText("", 4);
    user.SendText("<ansi fg=\"white\">The humming intensifies. The lights go out.</ansi>", 4);
    user.SendText("", 5);
    user.SendText("<ansi fg=\"red\">WARNING: NEURAL HANDSHAKE DETECTED</ansi>", 5);
    user.SendText("<ansi fg=\"red\">WARNING: REALITY ANCHOR FAILING</ansi>", 6);
    user.SendText("", 7);
    user.SendText("<ansi fg=\"magenta\">The lime on the machine begins to glow...</ansi>", 7);
    user.SendText("", 8);
    user.SendText("<ansi fg=\"cyan-hi\">╔══════════════════════════════════════════════════════════╗</ansi>", 8);
    user.SendText("<ansi fg=\"cyan-hi\">║  T R A N S F E R R I N G   C O N S C I O U S N E S S    ║</ansi>", 9);
    user.SendText("<ansi fg=\"cyan-hi\">╚══════════════════════════════════════════════════════════╝</ansi>", 10);
    user.SendText("", 11);
    user.SendText("<ansi fg=\"green\">You feel yourself falling through layers of code...</ansi>", 11);
    user.SendText("<ansi fg=\"green\">Through decades of digital sediment...</ansi>", 12);
    user.SendText("<ansi fg=\"green\">Through the membrane between worlds...</ansi>", 13);
    user.SendText("", 14);
    
    room.SendText(
        "<ansi fg=\"magenta\">" + user.GetCharacterName(false) + " dissolves into pixels and vanishes into the old machine!</ansi>",
        user.UserId(),
        15
    );
    
    // Move the player
    user.MoveRoom(FROSTFANG_TOWN_SQUARE, 16);
    
    // Welcome message
    user.SendText("", 17);
    user.SendText("<ansi fg=\"cyan\">You materialize in a swirl of lime-green light.</ansi>", 17);
    user.SendText("<ansi fg=\"yellow\">The cold hits you first. Then the smell of snow and woodsmoke.</ansi>", 18);
    user.SendText("<ansi fg=\"white\">This isn't a game anymore. This is real.</ansi>", 19);
    user.SendText("<ansi fg=\"white\">Welcome to Frostfang.</ansi>", 20);
    user.SendText("", 21);
    
    // Give them the fantasy newbie kit
    newbieKit = CreateItem(NEWBIE_KIT);
    user.GiveItem(newbieKit);
    user.SendText("<ansi fg=\"lime\">A mysterious package materializes in your hands - gear for this strange new world.</ansi>", 22);
    user.SendText("<ansi fg=\"yellow\">Type: use newbie kit</ansi>", 23);
}

function onIdle(room) {
    round = UtilGetRoundNumber();
    
    if (round % 10 == 0) {
        room.SendText("<ansi fg=\"green\">The old machine's screen blinks: 'AWAITING AUTHENTICATION...'</ansi>");
        return true;
    }
    
    return false;
}
