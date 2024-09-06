/* showActive(req.active[0]);
 * showSide(req.side.pokemon); */
const socket = new WebSocket("ws://localhost:8080/ws");

socket.onopen = function (event) {
  console.log("WebSocket connected.");
  const intervalId = setInterval(() => {
    socket.send("u");
  }, 1000);
};

socket.onmessage = function (event) {
  if (event.data == "inactive") {
    document.getElementById("moves").innerHTML = "";
    document.getElementById("switch").innerHTML = "";
    document.getElementById("tera").innerHTML = "";
    document.getElementById("messages").innerHTML = "";
    document.getElementById("messages").innerHTML = "Please wait...";
    return;
  }
  if (event. data == "uperr") {
    document.getElementById("messages").innerHTML =
      "Error getting update from server.";
    return;
  }
  const recv = JSON.parse(event.data);
  if (Array.isArray(recv)) {
    document.getElementById("messages").innerHTML = event.data;
    return;
  }
  if (recv.wait) {
    return;
  }
  document.getElementById("moves").innerHTML = "";
  document.getElementById("switch").innerHTML = "";
  document.getElementById("tera").innerHTML = "";
  document.getElementById("messages").innerHTML = "";
  var must_switch = recv.forceSwitch?.[0];
  if (must_switch != true) {
    showActive(recv.active[0]);
  }
  showSide(recv.side.pokemon);
  console.log("Message from server:", event.data);
};

socket.onerror = function (error) {
  console.error("WebSocket Error:", error);
};

socket.onclose = function (event) {
  if (event.wasClean) {
    console.log("WebSocket closed cleanly");
  } else {
    console.error("WebSocket connection closed unexpectedly");
  }
};

function showActive(active) {
  var adiv = document.getElementById("moves");
  adiv.innerHTML = "";
  var i = 0;
  for (const move of active.moves) {
    var b = document.createElement("button");
    b.innerHTML = `${move.move}\n${move.pp}/${move.maxpp}`;
    b.addEventListener("click", makeVote(i, "move"));
    b.disabled = move.disabled;
    i++;
    adiv.appendChild(b);
  }
}

function showSide(side) {
  var sdiv = document.getElementById("switch");
  sdiv.innerHTML = "";
  var i = 0;
  for (const p of side) {
    var b = document.createElement("button");
    b.innerHTML = `${p.details} ${p.condition}`;
    if (p.active || p.condition === "0 fnt") {
      b.disabled = true;
    }
    b.addEventListener("click", makeVote(i, "switch"));
    i++;
    sdiv.appendChild(b);
  }
}

function makeVote(i, t) {
  return (e) => {
    socket.send(
      "v" +
        JSON.stringify({
          type: t,
          idx: i,
          tera: false,
        }),
    );
  };
}
