// 必须使用 HashRouter 路由，不能使用别的 Router
import { Routes, Route, Navigate, HashRouter as Router } from 'react-router-dom'
import { AppProvider } from '@/store/AppContext'
import PhoneLayout from '@/components/PhoneLayout'
import OwnerHome from '@/pages/OwnerHome'
import Publish from '@/pages/Publish'
import OwnerTask from '@/pages/OwnerTask'
import OwnerOrder from '@/pages/OwnerOrder'
import OwnerOrders from '@/pages/OwnerOrders'
import OwnerMe from '@/pages/OwnerMe'
import StudentHall from '@/pages/StudentHall'
import StudentTask from '@/pages/StudentTask'
import StudentMy from '@/pages/StudentMy'
import StudentFulfill from '@/pages/StudentFulfill'
import StudentEarnings from '@/pages/StudentEarnings'
import StudentVerify from '@/pages/StudentVerify'

function App() {
    return (
        <AppProvider>
            <Router>
                <Routes>
                    <Route element={<PhoneLayout />}>
                        <Route path="/" element={<Navigate to="/owner" replace />} />
                        <Route path="/owner" element={<OwnerHome />} />
                        <Route path="/owner/publish" element={<Publish />} />
                        <Route path="/owner/task/:id" element={<OwnerTask />} />
                        <Route path="/owner/order/:id" element={<OwnerOrder />} />
                        <Route path="/owner/orders" element={<OwnerOrders />} />
                        <Route path="/owner/me" element={<OwnerMe />} />
                        <Route path="/student" element={<StudentHall />} />
                        <Route path="/student/task/:id" element={<StudentTask />} />
                        <Route path="/student/my" element={<StudentMy />} />
                        <Route path="/student/fulfill/:id" element={<StudentFulfill />} />
                        <Route path="/student/earnings" element={<StudentEarnings />} />
                        <Route path="/student/verify" element={<StudentVerify />} />
                        <Route path="*" element={<Navigate to="/owner" replace />} />
                    </Route>
                </Routes>
            </Router>
        </AppProvider>
    )
}

export default App
