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
    
    // End of CackalackyCon story message
    user.SendText("", 17);
    user.SendText("<ansi fg=\"lime-bold\">═══════════════════════════════════════════════════════════</ansi>", 17);
    user.SendText("<ansi fg=\"yellow-bold\">        CONGRATULATIONS, HACKER!</ansi>", 18);
    user.SendText("<ansi fg=\"lime-bold\">═══════════════════════════════════════════════════════════</ansi>", 19);
    user.SendText("", 20);
    user.SendText("<ansi fg=\"white\">You've completed the CackalackyCon story!</ansi>", 20);
    user.SendText("<ansi fg=\"white\">You solved puzzles, collected fragments, and found the secret.</ansi>", 21);
    user.SendText("", 22);
    user.SendText("<ansi fg=\"cyan\">But this is just the beginning...</ansi>", 22);
    user.SendText("", 23);
    user.SendText("<ansi fg=\"yellow\">The machine has transported you to Frostfang — a full</ansi>", 23);
    user.SendText("<ansi fg=\"yellow\">fantasy MUD with quests, combat, skills, and adventure.</ansi>", 24);
    user.SendText("", 25);
    user.SendText("<ansi fg=\"white\">Explore. Fight. Level up. The world is yours.</ansi>", 25);
    user.SendText("<ansi fg=\"lime-bold\">═══════════════════════════════════════════════════════════</ansi>", 26);
    user.SendText("", 27);
    
    // Give them the fantasy newbie kit
    newbieKit = CreateItem(NEWBIE_KIT);
    user.GiveItem(newbieKit);
    user.SendText("<ansi fg=\"lime\">A mysterious package materializes in your hands - gear for this strange new world.</ansi>", 28);
    user.SendText("<ansi fg=\"yellow\">Type: use newbie kit</ansi>", 29);
}

// FBI Raid settings
const REG_DESK_ROOM = 2002;
const RAID_CHANCE_PERCENT = 2;
const CHECK_INTERVAL_ROUNDS = 15;

function onIdle(room) {
    round = UtilGetRoundNumber();
    
    // FBI raid check
    if (round % CHECK_INTERVAL_ROUNDS == 0) {
        if (doFbiRaidCheck(room)) {
            return true;
        }
    }
    
    // Normal idle messages
    if (round % 10 == 0) {
        room.SendText("<ansi fg=\"green\">The old machine's screen blinks: 'AWAITING AUTHENTICATION...'</ansi>");
        return true;
    }
    
    return false;
}

function doFbiRaidCheck(room) {
    var players = room.GetPlayers();
    if (players.length == 0) {
        return false;
    }
    
    if (UtilDiceRoll(1, 100) > RAID_CHANCE_PERCENT) {
        return false;
    }
    
    var target = null;
    for (var i = 0; i < players.length; i++) {
        var player = players[i];
        if (player.GetRace().toLowerCase() == "fbi agent") {
            continue;
        }
        target = player;
        break;
    }
    
    if (target == null) {
        return false;
    }
    
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red-bold\">═══════════════════════════════════════════</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red-bold\">           🚨 FBI RAID! 🚨</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red-bold\">═══════════════════════════════════════════</ansi>");
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"yellow\">The door EXPLODES inward!</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">Federal agents swarm in, flashlights blazing!</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.SendText("<ansi fg=\"red\">\"FREEZE! FBI! GET ON THE GROUND!\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">An agent tackles you before you can reach the exit!</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"We got a live one! Take 'em back to registration!\"</ansi>");
    target.SendText("");
    
    SendRoomMessage(room.RoomId(), "<ansi fg=\"username\">" + target.GetCharacterName(false) + "</ansi> gets tackled by federal agents!");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">The feds drag them away toward registration...</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.MoveRoom(REG_DESK_ROOM);
    
    target.SendText("");
    target.SendText("<ansi fg=\"yellow\">The agents dump you unceremoniously at the registration desk.</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"Stay out of restricted areas, hacker.\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">They vanish as quickly as they appeared.</ansi>");
    target.SendText("");
    
    return true;
}
