// Hardware Hacking Village - Wire Connection Puzzle
// Player must connect wires in the correct order to unlock the fragment

const HARDWARE_FRAGMENT = 5101;

// The correct wire sequence (colors)
const CORRECT_SEQUENCE = ["green", "yellow", "red"];

function onCommand_connect(rest, user, room) {
    if (user.HasItemId(HARDWARE_FRAGMENT)) {
        user.SendText("<ansi fg=\"yellow\">You've already solved the hardware puzzle and have the fragment.</ansi>");
        return true;
    }
    
    var input = rest.toLowerCase().trim();
    var wires = input.split(/[\s,]+/);
    
    // Clean up wire names
    var cleanWires = [];
    for (var i = 0; i < wires.length; i++) {
        var w = wires[i].trim();
        if (w.length > 0) {
            cleanWires.push(w);
        }
    }
    
    if (cleanWires.length == 0) {
        user.SendText("<ansi fg=\"yellow\">Connect which wires? Example: connect green yellow red</ansi>");
        return true;
    }
    
    if (cleanWires.length != CORRECT_SEQUENCE.length) {
        user.SendText("");
        user.SendText("<ansi fg=\"red\">*BZZT* Wrong number of connections!</ansi>");
        user.SendText("<ansi fg=\"yellow\">The board has " + CORRECT_SEQUENCE.length + " connection points.</ansi>");
        user.SendText("");
        return true;
    }
    
    // Check the sequence
    var correct = true;
    for (var i = 0; i < CORRECT_SEQUENCE.length; i++) {
        if (cleanWires[i] != CORRECT_SEQUENCE[i]) {
            correct = false;
            break;
        }
    }
    
    if (!correct) {
        // Track attempts for hints
        var attempts = user.GetMiscData("hw_attempts");
        if (attempts == null) attempts = 0;
        attempts++;
        user.SetMiscData("hw_attempts", attempts);
        
        user.SendText("");
        user.SendText("<ansi fg=\"red\">*SPARK* *POP*</ansi>");
        user.SendText("<ansi fg=\"yellow\">The board smokes briefly. Wrong sequence.</ansi>");
        
        if (attempts >= 3) {
            user.SendText("");
            user.SendText("<ansi fg=\"cyan\">s0lray glances over: \"Lime green first. Always start with lime.\"</ansi>");
        }
        if (attempts >= 5) {
            user.SendText("<ansi fg=\"cyan\">\"Then yellow for caution, red for power.\"</ansi>");
        }
        
        user.SendText("");
        return true;
    }
    
    // SUCCESS!
    user.SendText("");
    user.SendText("<ansi fg=\"green\">*CLICK* *CLICK* *CLICK*</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"lime\">The connections lock into place!</ansi>");
    user.SendText("<ansi fg=\"green\">LEDs ripple across the board in sequence.</ansi>");
    user.SendText("<ansi fg=\"cyan\">A hidden compartment slides open.</ansi>");
    user.SendText("");
    
    var fragment = CreateItem(HARDWARE_FRAGMENT);
    user.GiveItem(fragment);
    
    user.SendText("<ansi fg=\"lime\">You received: hardware key fragment</ansi>");
    user.SendText("");
    
    room.SendText(
        "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> completes the circuit! Something clicks open.",
        user.UserId()
    );
    
    // Clear their attempts
    user.SetMiscData("hw_attempts", null);
    
    return true;
}

function onCommand_examine(rest, user, room) {
    var target = rest.toLowerCase().trim();
    
    if (target.includes("board") || target.includes("puzzle") || 
        target.includes("wire") || target.includes("circuit")) {
        showPuzzle(user);
        return true;
    }
    
    if (target.includes("bin") || target.includes("component")) {
        user.SendText("");
        user.SendText("<ansi fg=\"white\">Bins of components: resistors, capacitors, LEDs...</ansi>");
        user.SendText("<ansi fg=\"white\">You spot wires in three colors: green, yellow, and red.</ansi>");
        user.SendText("<ansi fg=\"cyan\">A sticky note reads: \"Lime first, always.\"</ansi>");
        user.SendText("");
        return true;
    }
    
    return false;
}

function onCommand_look(rest, user, room) {
    var target = rest.toLowerCase().trim();
    
    if (target.includes("board") || target.includes("puzzle") || target.includes("circuit")) {
        showPuzzle(user);
        return true;
    }
    
    return false;
}

function showPuzzle(user) {
    if (user.HasItemId(HARDWARE_FRAGMENT)) {
        user.SendText("<ansi fg=\"yellow\">The board is already solved. The compartment stands open.</ansi>");
        return;
    }
    
    user.SendText("");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("<ansi fg=\"yellow\">      HARDWARE CHALLENGE BOARD</ansi>");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"white\">A prototype board with three connection points.</ansi>");
    user.SendText("<ansi fg=\"white\">Three colored wires dangle nearby:</ansi>");
    user.SendText("");
    user.SendText("  <ansi fg=\"green\">● GREEN</ansi>  <ansi fg=\"yellow\">● YELLOW</ansi>  <ansi fg=\"red\">● RED</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"white\">A label reads: \"Connect in sequence to unlock.\"</ansi>");
    user.SendText("<ansi fg=\"white\">Below it, someone scratched: \"GND → CAUTION → PWR\"</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"yellow\">Type: connect [color] [color] [color]</ansi>");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
}

function onEnter(user, room) {
    if (user.HasItemId(HARDWARE_FRAGMENT)) {
        return false;
    }
    
    // Hint for new arrivals
    var round = UtilGetRoundNumber();
    if (round % 5 == 0) {
        user.SendText("<ansi fg=\"cyan\">s0lray looks up from a microscope. \"Try the challenge board. Examine it.\"</ansi>");
    }
    return false;
}

// Add puzzle hint to room nouns
function onCommand_read(rest, user, room) {
    var target = rest.toLowerCase().trim();
    if (target.includes("note") || target.includes("sticky") || target.includes("label")) {
        user.SendText("");
        user.SendText("<ansi fg=\"white\">The sticky note reads:</ansi>");
        user.SendText("<ansi fg=\"lime\">\"When in doubt, start with lime (green).\"</ansi>");
        user.SendText("<ansi fg=\"yellow\">\"Yellow means proceed with caution.\"</ansi>");
        user.SendText("<ansi fg=\"red\">\"Red brings the power.\"</ansi>");
        user.SendText("");
        return true;
    }
    return false;
}
