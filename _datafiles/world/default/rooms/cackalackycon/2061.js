// Practice Booth - Brass Padlock (Medium)
const BOOTH_KEY = "lockpick_booth_2061";
const BOOTH_EXIT_REWARD_SUFFIX = "_entered_reward_room";
const VILLAGE_ROOM = 2020;

function onEnter(user, room) {
    user.SetMiscCharacterData(BOOTH_KEY + "_unlocked", true);
    return true;
}

function onCommand(cmd, rest, user, room) {
    var target = (rest || "").toLowerCase().trim();
    var exitTarget = (cmd == "go") ? target : cmd.toLowerCase();
    if (exitTarget == "out") {
        user.SetTempData(BOOTH_KEY + BOOTH_EXIT_REWARD_SUFFIX, true);
        user.MoveRoom(VILLAGE_ROOM);
        return true;
    }
    return false;
}
