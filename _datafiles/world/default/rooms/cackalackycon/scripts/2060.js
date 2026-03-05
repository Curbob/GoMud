// Practice Booth - Clear Lock (Easy)
// Sets pending reward flag on entry

const BOOTH_KEY = "lockpick_booth_2060";

function onEnter(user, room) {
    // Mark that they entered this booth (pending reward if they escape)
    user.SetTempData(BOOTH_KEY, true);
    
    if (user.GetMiscData(BOOTH_KEY + "_completed")) {
        user.SendText("<ansi fg=\"yellow\">You've mastered this lock. The exit is easy now.</ansi>");
    } else {
        user.SendText("<ansi fg=\"green\">Pick the exit lock to escape and earn 5 gold!</ansi>");
    }
    return false;
}
