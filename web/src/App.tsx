import { useState } from 'react'
import reactLogo from './assets/react.svg'
import './App.css'
import { render } from 'react-dom'
import React, { Component } from 'react';
import NavbarScroller from './components/NavbarScroller';

const navigation = {
  brand: { name: "Pønskelisten", to: "/" },
  links: [
    { name: "Groups", to: "/groups" },
    { name: "Log in", to: "/login" },
    { name: "My account", to: "/account" }
  ]
}

class App extends Component {
  render() {
    // Descructured object for cleaner code :-)
    const { brand, links } = navigation;

    return (
      <div className="App">
        <NavbarScroller brand={brand} links={links} />
      </div>
    );
  }
}

export default App
