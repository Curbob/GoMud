// Cipher Room - Crypto fragment puzzle
// Player must solve a riddle to get the fragment

const CRYPTO_FRAGMENT = 5100;

function onCommand_solve(rest, user, room) {
    // Check if they already have it
    if (user.HasItemId(CRYPTO_FRAGMENT)) {
        user.SendText("<ansi fg=\"yellow\">You've already solved the cipher and have the crypto fragment.</ansi>");
        return true;
    }
    
    // The answer is "lime" (con theme!)
    answer = rest.toLowerCase().trim();
    if (answer == "lime" || answer == "a lime" || answer == "limes") {
        user.SendText("");
        user.SendText("<ansi fg=\"green\">CORRECT!</ansi>");
        user.SendText("");
        user.SendText("<ansi fg=\"cyan\">The cipher wheel clicks into place. A hidden compartment opens,</ansi>");
        user.SendText("<ansi fg=\"cyan\">revealing a glowing fragment etched with cryptographic symbols.</ansi>");
        user.SendText("");
        
        fragment = CreateItem(CRYPTO_FRAGMENT);
        user.GiveItem(fragment);
        
        user.SendText("<ansi fg=\"lime\">You received: crypto key fragment</ansi>");
        user.SendText("");
        
        room.SendText(
            "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> solves the cipher! A compartment opens with a soft click.",
            user.UserId()
        );
        return true;
    }
    
    user.SendText("<ansi fg=\"red\">The cipher wheel buzzes. Incorrect.</ansi>");
    return true;
}

function onCommand_read(rest, user, room) {
    if (rest.toLowerCase().includes("cipher") || 
        rest.toLowerCase().includes("puzzle") ||
        rest.toLowerCase().includes("riddle") ||
        rest.toLowerCase().includes("wheel")) {
        showPuzzle(user);
        return true;
    }
    return false;
}

function onCommand_examine(rest, user, room) {
    if (rest.toLowerCase().includes("cipher") || 
        rest.toLowerCase().includes("puzzle") ||
        rest.toLowerCase().includes("wheel")) {
        showPuzzle(user);
        return true;
    }
    return false;
}

function showPuzzle(user) {
    user.SendText("");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("<ansi fg=\"yellow\">        THE CIPHER WHEEL PUZZLE</ansi>");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"white\">Decode this message:</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"green\">  YVNR</ansi>");
    user.SendText("<ansi fg=\"white\">  (ROT13 encoded)</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"yellow\">Hint: It's green, it's a fruit, and it's</ansi>");
    user.SendText("<ansi fg=\"yellow\">      EVERYWHERE at this con.</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"white\">Type: solve [answer]</ansi>");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
}
