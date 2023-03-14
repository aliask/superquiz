'use strict';

const express = require('express');

// Constants
const PORT = 8080;
const HOST = '0.0.0.0';

// App
const app = express();
const router = express.Router();

router.get('/', function(req,res){
    res.sendFile('public/index.html');
});

router.get('/smh', async function(req, res, next) {
    try {
        const apiResponse = await (await fetch(req.query.q)).text()
        res.send(apiResponse)
    } catch (err) {
        console.log(err)
        res.status(500).send('Something went wrong')
    }
})

app.use(express.static('public'))
app.use('/', router);

app.listen(PORT, HOST, () => {
    console.log(`Running on http://${HOST}:${PORT}`);
});