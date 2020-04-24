var allJobs = [];
var pendingJobs = [];
var completedJobs = [];
/** Add a log item. `id` is optional and if supplied will set the id of the dom element created. */
function addLogItem(text, id) {
    var doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
    var item = document.createElement("div");
    item.innerText = text;
    if (id) {
        item.id = id;
    }
    log.appendChild(item);
    if (doScroll) {
        log.scrollTop = log.scrollHeight - log.clientHeight;
    }
}
/** Update a log item by id. `addClass` is optional and, if supplied will be added to the log item classlist. */
function updateLogItem(text, id, addClass) {
    var itemToUpdate = document.getElementById(id);
    if (addClass) {
        itemToUpdate.classList.add(addClass);
    }
    itemToUpdate.innerText = text;
}
var conn;
/** Establish websocket connection to jobMonitor endpoint. If we have any pending jobs, will immediately check their status. */
function connectToWebsocket() {
    conn = new WebSocket("ws://" + document.location.host + "/jobMonitor");
    conn.onclose = function (evt) { addLogItem('Connection closed.'); };
    conn.onmessage = function (evt) {
        var jobID = evt.data;
        pendingJobs.splice(pendingJobs.indexOf(jobID), 1);
        completedJobs.push(jobID);
        jobNo = allJobs.indexOf(jobID) + 1;
        updateLogItem(`Job #${jobNo} (${jobID}) - complete.`, jobID, 'complete');
        setCounts();
    };
    conn.onopen = function (event) {
        addLogItem('Connection opened.');
        // If we have pending jobs, check their status
        if (pendingJobs.length > 0) {
            // Create copy of array since it will be modified asynchronously by conn.onmessage
            jobsToCheck = [...pendingJobs];
            for (var i = 0; i < jobsToCheck.length; i++) {
                addJobToCheckStatus(jobsToCheck[i]);
            }
        }
    };
}
/** Disconnect from our websocket. */
function disconnect() {
    conn.close();
    conn = null;
}
/** Create a job. On success, will add to `pendingJobs` array and start monitoring. */
function createJob() {
    httpRequest = new XMLHttpRequest();
    httpRequest.open('GET', 'createJob');
    httpRequest.onreadystatechange = function () {
        if (httpRequest.readyState === XMLHttpRequest.DONE) {
            jobID = httpRequest.responseText
            pendingJobs.push(jobID);
            allJobs.push(jobID);
            addLogItem(`Job #${allJobs.length} (${jobID}) - running...`, jobID);
            addJobToCheckStatus(jobID);
            setCounts();
        }
    }
    httpRequest.send();
}
/** Let our webhook connection know we want to start monitoring a job by ID. */
function addJobToCheckStatus(jobID) {
    if (!conn || conn.readyState != WebSocket.OPEN) {
        return;
    }
    conn.send(jobID);
}
/** This is basically a "tick" for our count UI. Sets count values. */
function setCounts() {
    pending.innerText = pendingJobs.length;
    completed.innerText = completedJobs.length;
    total.innerText = allJobs.length;
}
// Variables for our html elements that we will get on window.onload.
var msg, log, pending, completed, total;
window.onload = function () {
    msg = document.getElementById("msg");
    log = document.getElementById("log");
    pending = document.getElementById("pending");
    completed = document.getElementById("completed");
    total = document.getElementById("total");
};