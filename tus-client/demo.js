/* global tus */
/* eslint no-console: 0 */

"use strict";

var upload          = null;
var uploadIsRunning = false;
var toggleBtn       = document.querySelector("#toggle-btn");
var resumeCheckbox  = document.querySelector("#resume");
var input           = document.querySelector("input[type=file]");
var progress        = document.querySelector(".progress");
var progressBar     = progress.querySelector(".bar");
var alertBox        = document.querySelector("#support-alert");
var uploadList      = document.querySelector("#upload-list");
var chunkInput      = document.querySelector("#chunksize");
var endpointInput   = document.querySelector("#endpoint");
var token = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJYQXhWQTVENGtDZm1nZzF3TDNsVkk3Nng0YXJqb1ZwVkFISkVaMHRVMFlzIn0.eyJleHAiOjE2NDY5MTEwOTksImlhdCI6MTY0Njg4MjI5OSwianRpIjoiYjAwZmEyNTItOWU4OC00NjBmLWJjMGYtZWU2MTE0MDVjZGU0IiwiaXNzIjoiaHR0cHM6Ly9rZXljbG9hay5jbG91ZG9reW8uZGV2L2F1dGgvcmVhbG1zL3ZvZGNsb3VkIiwic3ViIjoiYzhmZTcwZmEtMmM3Yi00MTBlLTkwOGMtOGEyZTMzYmM0MDA3IiwidHlwIjoiQmVhcmVyIiwiYXpwIjoiYWJjLWJhY2tlbmQiLCJzZXNzaW9uX3N0YXRlIjoiNzAwODQ1MTktMGZkYS00YzViLWE4YmEtOTQyOGU3YmVkMjc5IiwiYWNyIjoiMSIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ2b2Rfc2VydmVyIl19LCJzY29wZSI6InByb2ZpbGUiLCJzaWQiOiI3MDA4NDUxOS0wZmRhLTRjNWItYThiYS05NDI4ZTdiZWQyNzkiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ2b2QtdGFpLW5ndXllbiJ9.mqLtRh4a1chesdIrbeo1FMrcTuMNgrMskxxx3TqGYqJe284k-8PWk0EwWReVj-BU2eixtsorDY8kTOI0zgXvwQY01NxIgeQ1SWa2H7_prQFXNHQzrAYRRyX0EVFe7xG0KtfSYmqxg8Z8v_AV2cFl4_1IHnWoLElsE72Qnn5rzRx9eZ6JbbRSBOgCcozB3xTClrDOG5K5LIRqD16Qp0XpeyoclJoiMlAwMbSvk6oizbPUzvxxQc2ucqFNkT54MGFPhzmaYfxIdXGXLImwo1uwD4yJK2PZBNREf2QQW7Dwu6x3bAB5CT3kB1JKUC2GL-4iH7qomrQorDMspFrVYTToPA"  
// var token = "Bearer FalseJWT"
if (!tus.isSupported) {
  alertBox.classList.remove("hidden");
}

if (!toggleBtn) {
  throw new Error("Toggle button not found on this page. Aborting upload-demo. ");
}

toggleBtn.addEventListener("click", function (e) {
  e.preventDefault();

  if (upload) {
    if (uploadIsRunning) {
      upload.abort();
      toggleBtn.textContent = "resume upload";
      uploadIsRunning = false;
    } else {
      upload.start();
      toggleBtn.textContent = "pause upload";
      uploadIsRunning = true;
    }
  } else {
    if (input.files.length > 0) {
      startUpload();
    } else {
      input.click();
    }
  }
});

input.addEventListener("change", startUpload);

function startUpload() {
  var file = input.files[0];
  // Only continue if a file has actually been selected.
  // IE will trigger a change event even if we reset the input element
  // using reset() and we do not want to blow up later.
  if (!file) {
    return;
  }

  console.log(file)
  var endpoint = endpointInput.value;
  var chunkSize = parseInt(chunkInput.value, 10);
  if (isNaN(chunkSize)) {
    chunkSize = Infinity;
  }

  toggleBtn.textContent = "pause upload";

  var options = {
    endpoint: endpoint,
    resume  : !resumeCheckbox.checked,
    chunkSize: chunkSize,
    headers: {
      Authorization: token
    },
    metadata: {
      filename: file.name,
      filetype: file.type,
      parent_folders: "/vod/test/video/",
      product_id: 1234,
      language: "en",
      preview_start:"00:00:00",
      preview_end:"00:00:20"
    },
    retryDelays: [0, 1000, 3000, 5000],
    onError : function (error) {
      if (error.originalRequest) {
        if (window.confirm("Failed because: " + error + "\nDo you want to retry?")) {
          upload.start();
          uploadIsRunning = true;
          return;
        }
      } else {
        window.alert("Failed because: " + error);
      }

      reset();
    },
    onProgress: function (bytesUploaded, bytesTotal) {
      var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2);
      progressBar.style.width = percentage + "%";
      console.log(bytesUploaded, bytesTotal, percentage + "%");
    },
    onSuccess: function () {
      var anchor = document.createElement("a");
      anchor.textContent = "Download " + upload.file.name + " (" + upload.file.size + " bytes)";
      // anchor.href = upload.url;
      // anchor.target = "_blank";
      anchor.onclick = download1(upload.url);
      anchor.className = "btn btn-success";
      // anchor.setAttribute("onclick","download1('"+upload.url+"')");
      uploadList.appendChild(anchor);
      reset();
    }
  };

  upload = new tus.Upload(file, options);
  upload.start();
  uploadIsRunning = true;
}

function reset() {
  input.value = "";
  toggleBtn.textContent = "start upload";
  upload = null;
  uploadIsRunning = false;
}

function download1(url){
  const xhttp = new XMLHttpRequest();
  xhttp.onload = function() {
    // document.getElementById("demo").innerHTML = this.responseText;
    console.log("Response ")
    console.log(this.responseText)
  }
  xhttp.open("GET", url);
  xhttp.setRequestHeader("Authorization", token);
  xhttp.send();
}