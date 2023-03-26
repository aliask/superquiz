const loadButton = document.querySelector('#load-button')
const urlTextbox = document.querySelector('#url-textbox')

function findJSONFile(payload) {
    let re = /https:\/\/www.smh.com.au\/interactive\/hub\/configs\/[a-z-]+quiz\/\d+.json/gi
    regex = new RegExp(re)
    matches = payload.match(regex)

    if(matches) {
        return matches[0]
    } else {
        return null
    }
}

function updatePoints() {
    let score = 0
    document.querySelectorAll(".correct").forEach(item => {
        score += parseInt(item.querySelector(".btn-outline-success").dataset.points)
    })
    document.querySelector("#score").innerHTML = `${score} point${score !== 1 ? 's' : ''}`
}

function right(button) {
    party.confetti(button)
    button.parentElement.parentElement.parentElement.parentElement.classList.add("correct")
    button.parentElement.parentElement.parentElement.parentElement.classList.remove("incorrect")
    button.parentElement.parentElement.parentElement.parentElement.classList.remove("unanswered")
    button.parentElement.querySelector(".btn-outline-danger").classList.remove("active")
    button.classList.add("active")
    updatePoints()
}

function wrong(button) {
    button.parentElement.parentElement.parentElement.parentElement.classList.add("incorrect")
    button.parentElement.parentElement.parentElement.parentElement.classList.remove("correct")
    button.parentElement.parentElement.parentElement.parentElement.classList.remove("unanswered")
    button.parentElement.querySelector(".btn-outline-success").classList.remove("active")
    button.classList.add("active")
    updatePoints()
}

function display_question(question, index, difficulty) {
    switch(difficulty) {
        case "Easy":
            points = 1
            break
        case "Medium":
            points = 2
            break
        case "Hard":
            points = 3
            break
    }
    return `<div class="accordion-item unanswered">
        <h2 class="accordion-header" id="heading${difficulty}${index}">
            <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#collapse${difficulty}${index}" aria-expanded="false" aria-controls="collapse${difficulty}${index}">${question.question}</button>
        </h2>
        <div id="collapse${difficulty}${index}" class="accordion-collapse collapse" aria-labelledby="heading${difficulty}${index}" data-bs-parent="#accordion${difficulty}">
        <div class="accordion-body">
            ${question.answer}
            <div class="text-end">
                <button class="btn btn-outline-success ohyeah" data-points="${points}" onclick="right(this)">I got it right!</button>
                <button class="btn btn-outline-danger ohno" onclick="wrong(this)">I got it wrong!</button>
            </div>
        </div>
        </div>
    </div>`
}

function displayQuiz(json) {
    easy = json.data.filter(item => item.level == 1)
    medium = json.data.filter(item => item.level == 2)
    hard = json.data.filter(item => item.level == 3)

    response = `<h2>${json.config.name}</h2>`
    // If the level is not set, we just display all questions
    if(easy.length == 0) {
        response += `<div class="accordion" id="accordionEasy">`
        json.data.forEach((question, index) => {
            response += display_question(question, index, "Easy")
        });
        response += `</div><hr>`
    } else { // Otherwise, we display the questions by level
        response += `<h3>Easy - 1 point</h3>`
        response += `<div class="accordion" id="accordionEasy">`
        easy.forEach((question, index) => {
            response += display_question(question, index, "Easy")
        })
        response += `</div><hr>`

        response += `<h3>Medium - 2 points</h3>`
        response += `<div class="accordion" id="accordionMedium">`
        medium.forEach((question, index) => {
            response += display_question(question, index, "Medium")
        })
        response += `</div><hr>`

        response += `<h3>Hard - 3 points</h3>`
        response += `<div class="accordion" id="accordionHard">`
        hard.forEach((question, index) => {
            response += display_question(question, index, "Hard")
        })
        response += `</div><hr>`
    }
    document.querySelector("#quizContent").innerHTML = response
    updatePoints()
}

function loadQuiz(url) {
    fetch('/smh?q=' + encodeURIComponent(url))
        .then(response => response.text())
        .then(data => {
            jsonFile = findJSONFile(data)
            if(jsonFile) {
                fetch('/smh?q=' + encodeURIComponent(jsonFile))
                .then(response => response.json())
                .then(displayQuiz)
            } else {
                alert("Couldn't find Quiz data on requested page.")
            }
        })
}

loadButton.addEventListener('click', event => {
    loadQuiz(urlTextbox.value)
})

function loadCheck() {
    let params = new URL(window.location.href)
    q = params.searchParams.get("q")
    if(q) {
        urlTextbox.value = q
        window.setTimeout(loadQuiz, 0, urlTextbox.value)
    }
}
