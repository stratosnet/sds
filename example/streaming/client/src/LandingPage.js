import React from 'react';
import {Link, useNavigate} from "react-router-dom";
import myVideos from '../videos.json';
import "./style.css"
import videoLinkType from "./LinkType";
import LinkType from "./LinkType";

const LandingPage = () => {
    const [linkList, setLinkList] = React.useState(myVideos);
    const [customLinkType, setCustomLinkType] = React.useState(LinkType.FILE_HASH);
    const [customLink, setCustomLink] = React.useState("");
    const [sharePassword, setSharePassword] = React.useState("");
    const navigate = useNavigate();

    function handleCustomLinkClick() {
        let videoUrl = "";
        switch (customLinkType) {
            case LinkType.FILE_HASH:
                videoUrl = `/video/${customLinkType}/${customLink.trim()}`;
                break;
            case LinkType.SHARE_LINK:
                videoUrl = `/video/${customLinkType}/${customLink.trim()}?pw=${sharePassword.trim()}`;
                break;
            default:
                break;
        }
        if (videoUrl.length > 0) {
            navigate(videoUrl);
        }
    }

    React.useEffect(() => {
        async function fetchLinks() {
            const url = process.env.REACT_APP_NODE_SIGN_URL;
            if (!url) {
                return;
            }
            const resp = await fetch(`${url}/api/link-list`)
            const moreLinks = await resp.json()
            setLinkList([...linkList, ...moreLinks])
        }
        fetchLinks();
    }, [])

    return <div className="link-table">
        <table>
            <tbody>
            <tr>
                <th>Link Type</th>
                <th>Link</th>
                <th>File Name</th>
            </tr>
            {
                linkList.map(link => {
                    if (link.linkType === videoLinkType.FILE_HASH) {
                        return <tr key={link.fileHash}>
                            <td>
                                File Hash (owned file)
                            </td>
                            <td>
                                <Link to={`/video/${link.linkType}/${link.fileHash}`}>{link.fileHash}</Link>
                            </td>
                            <td>
                                {link.fileName}
                            </td>
                        </tr>
                    } else if (link.linkType === videoLinkType.SHARE_LINK) {
                        return <tr key={link.shareLink}>
                            <td>
                                Share Link
                            </td>
                            <td>
                                <Link
                                    to={`/video/${link.linkType}/${link.shareLink}?pw=${link.sharePassword}`}>{link.shareLink}</Link>
                            </td>
                            <td>
                                {link.fileName}
                            </td>
                        </tr>
                    }
                })
            }
            <tr style={{height: "50px"}}>
                <td>
                    <select name="linkType" onChange={e => setCustomLinkType(e.target.value)} value={customLinkType}>
                        <option value={LinkType.FILE_HASH}>File Hash</option>
                        <option value={LinkType.SHARE_LINK}>Share Link</option>
                    </select>
                </td>
                <td>
                    <div style={{display: "flex"}}>
                        Link <input size={36} name="link" value={customLink}
                                    onChange={e => setCustomLink(e.target.value)}></input>
                        {customLinkType === LinkType.SHARE_LINK &&
                            <div>Password <input size={5} name="sharePassword" value={sharePassword}
                                                 onChange={e => setSharePassword(e.target.value)}></input></div>}
                    </div>
                </td>
                <td>
                    <button disabled={customLink == null || customLink.length === 0}
                            onClick={() => handleCustomLinkClick()}>GO
                    </button>
                </td>
            </tr>
            </tbody>
        </table>
    </div>
};

export default LandingPage