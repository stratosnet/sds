import React from 'react';
import { BrowserRouter, Routes, Route } from "react-router-dom";
import StreamPlayer from "./StreamPlayer";
import LandingPage from "./LandingPage";
import KeyPairContextProvider from "./KeyPairContext";

const App = () => {
    return (
        <BrowserRouter>
            <KeyPairContextProvider>
                <Routes>
                    <Route index element={<LandingPage />} />
                    <Route path="video/:linkType/:link" element={<StreamPlayer />} />
                </Routes>
            </KeyPairContextProvider>
        </BrowserRouter>
    );
};

export default App;


