(function () {
  var track = document.querySelector(".slide-track");
  var dots = Array.prototype.slice.call(document.querySelectorAll(".nav-dot"));

  if (!track || dots.length === 0) {
    return;
  }

  var activeSlide = document.body.getAttribute("data-active-slide");
  var currentIndex = Math.max(0, ["energy", "planes", "weather"].indexOf(activeSlide));
  var interval;
  var delay = 30000;
  var pages = [
    { id: "slide-energy", url: "/energy", name: "energie", refreshDelay: 30000 },
    { id: "slide-planes", url: "/planes", name: "vliegtuigen", refreshDelay: 1500 },
    { id: "slide-weather", url: "/weather", name: "het weer", refreshDelay: 30000 }
  ];

  function loadPage(page) {
    var target = document.getElementById(page.id);

    return fetch(page.url, { cache: "no-store" })
      .then(function (response) {
        if (!response.ok) {
          throw new Error("Verzoek mislukt met status " + response.status);
        }
        return response.text();
      })
      .then(function (markup) {
        var now = window.performance ? window.performance.now() : Date.now();
        var rootStyle = document.documentElement.style;
        rootStyle.setProperty("--radar-sync", "-" + (now % 3200).toFixed(1) + "ms");
        rootStyle.setProperty("--pulse-sync", "-" + (now % 2000).toFixed(1) + "ms");
        target.innerHTML = markup;
      })
      .catch(function (error) {
        console.error("Kan dashboard voor " + page.name + " niet laden:", error);
        target.innerHTML = "";
        var message = document.createElement("p");
        message.className = "slide-error";
        message.textContent = "Kan de gegevens voor " + page.name + " niet laden.";
        target.appendChild(message);
      });
  }

  function updateClock() {
    var clock = document.querySelector(".clock");
    if (!clock) {
      return;
    }
    clock.textContent = new Date().toLocaleTimeString("nl-NL", {
      timeZone: "Europe/Amsterdam",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit"
    });
  }

  function refreshPage(page) {
    loadPage(page).then(function () {
      window.setTimeout(function () {
        refreshPage(page);
      }, page.refreshDelay);
    });
  }

  function showSlide(index) {
    currentIndex = (index + dots.length) % dots.length;
    track.style.transform = "translateX(-" + (currentIndex * 100 / dots.length) + "%)";

    dots.forEach(function (dot, dotIndex) {
      var isActive = dotIndex === currentIndex;
      dot.classList.toggle("active", isActive);
      dot.setAttribute("aria-current", isActive ? "true" : "false");
    });
  }

  function startRotation() {
    window.clearInterval(interval);
    interval = window.setInterval(function () {
      showSlide(currentIndex + 1);
    }, delay);
  }

  dots.forEach(function (dot, index) {
    dot.addEventListener("click", function () {
      showSlide(index);
      startRotation();
    });
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "ArrowRight") {
      showSlide(currentIndex + 1);
      startRotation();
    } else if (event.key === "ArrowLeft") {
      showSlide(currentIndex - 1);
      startRotation();
    }
  });

  updateClock();
  window.setInterval(updateClock, 1000);
  pages.forEach(refreshPage);
  showSlide(currentIndex);
  startRotation();
}());
