function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
    } else {
        var login_data = false;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="news_post">
                    
                            <div class="title">
                                Pønskelisten
                            </div>

                            <div class="body" style="text-align: center;">
                                Make a wish.
                            </div>

                            <br>
                            <br>

                            <div id="banner">
                            </div>
                            
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        info('Logged in.');
    } else {
        showLoggedOutMenu();
    }
}