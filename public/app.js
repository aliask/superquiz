const loadButton = document.querySelector('#load-button')
const urlTextbox = document.querySelector('#url-textbox')
const smhLink = document.querySelector('#smh-link')
const satPaperLink = document.querySelector('#sat-paper-link')

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
    // If there are no medium or hard questions, just display all questions without difficulty headers
    if(medium.length == 0 && hard.length == 0) {
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

function parseQuestionsFromDOM(html) {
    // Create a temporary DOM parser
    const parser = new DOMParser()
    const doc = parser.parseFromString(html, 'text/html')

    // Find all question and answer elements
    const questionElements = doc.querySelectorAll('.quiz-question__question')
    const answerElements = doc.querySelectorAll('.quiz-question__answer')

    // Check if we found any questions
    if(questionElements.length === 0) {
        return null
    }

    // Build question data array
    const questions = []
    questionElements.forEach((questionEl, index) => {
        if(answerElements[index]) {
            questions.push({
                question: questionEl.textContent.trim(),
                answer: answerElements[index].innerHTML.trim(),
                level: 1 // Default to level 1 (Easy) for DOM-parsed questions
            })
        }
    })

    // Return in the same format as the JSON API
    return {
        config: {
            name: doc.querySelector('title')?.textContent || 'Quiz'
        },
        data: questions
    }
}

function loadQuiz(url) {
    fetch('/smh?q=' + encodeURIComponent(url))
        .then(response => response.text())
        .then(data => {
            // First, try to find JSON file (existing behavior)
            jsonFile = findJSONFile(data)
            if(jsonFile) {
                fetch('/smh?q=' + encodeURIComponent(jsonFile))
                .then(response => response.json())
                .then(displayQuiz)
            } else {
                // If no JSON found, try parsing DOM elements
                const domQuiz = parseQuestionsFromDOM(data)
                if(domQuiz && domQuiz.data.length > 0) {
                    displayQuiz(domQuiz)
                } else {
                    alert("Couldn't find Quiz data on requested page.")
                }
            }
        })
}

const tagQuery = `
query TagIndexQuery(
  $brand: String!
  $count: Int!
  $offset: Int
  $render: Render
  $tagID: ID!
  $types: [AssetType!]
  $newsroom: NewsroomType
) {
  tag: tag(brand: $brand, id: $tagID, newsroom: $newsroom)
  {
    ...TagIndexFragment_tagData
    assetsConnection(
      brand: $brand,
      count: $count,
      offset: $offset,
      render: $render,
      types: $types
    ) {
      ...AssetsConnectionFragment_showMoreDataWithPrimaryAndSecondaryTags
    } id
  }
}

fragment TagIndexFragment_tagData on Tag {
  displayName
  id
}

fragment AssetsConnectionFragment_showMoreDataWithPrimaryAndSecondaryTags on AssetsConnection {
  assets {
    ...AssetFragmentFragment_assetDataWithPrimaryAndSecondaryTags
    id
  }
  pageInfo {
    endCursor
    hasNextPage
  }
}

fragment AssetFragmentFragment_assetDataWithPrimaryAndSecondaryTags on Asset {
  ...AssetFragmentFragment_assetData
  category {
    id
    name
  }
  tags {
    primary: primaryTag {
      ...AssetFragment_tagFragment
      id
    }
    secondary {
      ...AssetFragment_tagFragment
      id
    }
  }
}

fragment AssetFragmentFragment_assetData on Asset {
  id
  asset {
    about
    headlines {
      headline
    }
  }
  urls {
    canonical { path }
  }
  dates {
    modified
    published
  }
}

fragment AssetFragment_tagFragment on AssetTagDetails {
  displayName
}
`

async function fetchSMHArticle() {
  const smhGqlEndpoint = "https://api.smh.com.au/graphql"
  const variables = {
    "brand":"smh",
    "count":5,
    "newsroom":"METRO",
    "offset":0,
    "render":"WEB",
    "tagID":"6guy",
    "types": [
      "article"
    ]
  }
  let results = await fetch(smhGqlEndpoint, {
    method: 'POST',
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      query: tagQuery,
      variables
    })
  })
  let tags = await results.json();
  const path = tags.data.tag.assetsConnection.assets[0].urls.canonical.path
  return "https://www.smh.com.au" + path
}

async function fetchSaturdayPaperQuiz() {
  try {
    const response = await fetch('/smh?q=' + encodeURIComponent('https://www.thesaturdaypaper.com.au/quiz/'))
    const html = await response.text()

    // Parse the HTML
    const parser = new DOMParser()
    const doc = parser.parseFromString(html, 'text/html')

    // Look for quiz URLs in the format /quiz/2025/11/15
    const quizLinkRegex = /\/quiz\/\d{4}\/\d{2}\/\d{2}/g
    const links = Array.from(doc.querySelectorAll('a')).map(a => a.getAttribute('href'))
    const quizLinks = links.filter(link => link && quizLinkRegex.test(link))

    if(quizLinks.length > 0) {
      // Sort to get the most recent (assuming they're in chronological order in the URL)
      quizLinks.sort().reverse()
      return "https://www.thesaturdaypaper.com.au" + quizLinks[0]
    }

    return null
  } catch (err) {
    console.error('Error fetching Saturday Paper quiz:', err)
    return null
  }
}

loadButton.addEventListener('click', event => {
    loadQuiz(urlTextbox.value)
})

smhLink.addEventListener('click', async event => {
    event.preventDefault()
    const url = await fetchSMHArticle()
    if(url) {
        urlTextbox.value = url
        loadQuiz(url)
    } else {
        alert("Couldn't fetch latest SMH quiz")
    }
})

satPaperLink.addEventListener('click', async event => {
    event.preventDefault()
    const url = await fetchSaturdayPaperQuiz()
    if(url) {
        urlTextbox.value = url
        loadQuiz(url)
    } else {
        alert("Couldn't fetch latest Saturday Paper quiz")
    }
})

urlTextbox.addEventListener('focus', event => {
  urlTextbox.select();
})

async function loadCheck() {
    let params = new URL(window.location.href)
    q = params.searchParams.get("q")
    if(q) {
        urlTextbox.value = q
        window.setTimeout(loadQuiz, 0, urlTextbox.value)
    } else {
      const url = await fetchSMHArticle()
      if(url) {
          urlTextbox.value = url
      }
    }
}
